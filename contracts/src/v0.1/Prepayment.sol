// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "openzeppelin-solidity/contracts/access/AccessControlEnumerable.sol";
import "openzeppelin-solidity/contracts/access/Ownable.sol";

// import "./interfaces/BlockhashStoreInterface.sol";
// /* import "./interfaces/AggregatorV3Interface.sol"; */
// import "./interfaces/VRFCoordinatorInterface.sol";
// import "./interfaces/TypeAndVersionInterface.sol";
// import "./libraries/VRF.sol";
// import "./ConfirmedOwner.sol";
// import "./VRFConsumerBase.sol";

contract Subscriptions is Ownable, AccessControlEnumerable {
    uint16 public constant MAX_CONSUMERS = 100;
    uint96 private s_totalBalance;
    error TooManyConsumers();
    error InsufficientBalance();
    error InvalidConsumer(uint64 subId, address consumer);
    error InvalidSubscription();
    error OnlyCallableFromLink();
    error InvalidCalldata();
    error MustBeSubOwner(address owner);
    error PendingRequestExists();
    error MustBeRequestedOwner(address proposedOwner);
    error BalanceInvariantViolated(uint256 internalBalance, uint256 externalBalance); // Should never happen
    error Reentrant();

    event SubscriptionCreated(uint64 indexed subId, address owner);
    event SubscriptionFunded(uint64 indexed subId, uint256 oldBalance, uint256 newBalance);
    event SubscriptionConsumerAdded(uint64 indexed subId, address consumer);
    event SubscriptionConsumerRemoved(uint64 indexed subId, address consumer);
    event SubscriptionCanceled(uint64 indexed subId, address to, uint256 amount);
    event SubscriptionOwnerTransferRequested(uint64 indexed subId, address from, address to);
    event SubscriptionOwnerTransferred(uint64 indexed subId, address from, address to);
    event FundsRecovered(address to, uint256 amount);
    event ConfigSet(
        uint16 minimumRequestConfirmations,
        uint32 maxGasLimit,
        uint32 gasAfterPaymentCalculation,
        FeeConfig feeConfig
    );
    bytes32 public constant WITHDRAWER_ROLE = keccak256("WITHDRAWER_ROLE");
    
    constructor() {
        _setupRole(DEFAULT_ADMIN_ROLE, _msgSender());
    }

    struct Subscription {
        // There are only 1e9*1e18 = 1e27 juels in existence, so the balance can fit in uint96 (2^96 ~ 7e28)
        uint96 balance; // Common link balance used for all consumer requests.
        uint64 reqCount; // For fee tiers
    }
    struct SubscriptionConfig {
        address owner; // Owner can fund/withdraw/cancel the sub.
        address requestedOwner; // For safely transferring sub ownership.
        // Maintains the list of keys in s_consumers.
        // We do this for 2 reasons:
        // 1. To be able to clean up all keys from s_consumers when canceling a subscription.
        // 2. To be able to return the list of all consumers in getSubscription.
        // Note that we need the s_consumers map to be able to directly check if a
        // consumer is valid without reading all the consumers from storage.
        address[] consumers;
    }

    struct FeeConfig {
        // Flat fee charged per fulfillment in millionths of link
        // So fee range is [0, 2^32/10^6].
        uint32 fulfillmentFlatFeeLinkPPMTier1;
        uint32 fulfillmentFlatFeeLinkPPMTier2;
        uint32 fulfillmentFlatFeeLinkPPMTier3;
        uint32 fulfillmentFlatFeeLinkPPMTier4;
        uint32 fulfillmentFlatFeeLinkPPMTier5;
        uint24 reqsForTier2;
        uint24 reqsForTier3;
        uint24 reqsForTier4;
        uint24 reqsForTier5;
    }


    FeeConfig private s_feeConfig;
    mapping(address => mapping(uint64 => uint64)) private s_consumers;

    /* subId */ /* subscriptionConfig */
    mapping(uint64 => SubscriptionConfig) private s_subscriptionConfigs;

    /* subId */ /* subscription */
    mapping(uint64 => Subscription) private s_subscriptions;

    // We make the sub count public so that its possible to
    // get all the current subscriptions via getSubscription.
    uint64 private s_currentSubId;
    /* oracle */ /* KLAY balance */
    mapping(address => uint96) private s_withdrawableTokens;

    function getTotalBalance() external view returns (uint256) {
        return s_totalBalance;
    }


    function getSubscription(
        uint64 subId
    )
        external
        view
        returns (uint96 balance, uint64 reqCount, address owner, address[] memory consumers)
    {
        if (s_subscriptionConfigs[subId].owner == address(0)) {
            revert InvalidSubscription();
        }
        return (
            s_subscriptions[subId].balance,
            s_subscriptions[subId].reqCount,
            s_subscriptionConfigs[subId].owner,
            s_subscriptionConfigs[subId].consumers
        );
    }

    function createSubscription() external nonReentrant returns (uint64) {
        s_currentSubId++;
        uint64 currentSubId = s_currentSubId;
        address[] memory consumers = new address[](0);
        s_subscriptions[currentSubId] = Subscription({balance: 0, reqCount: 0});
        s_subscriptionConfigs[currentSubId] = SubscriptionConfig({
            owner: msg.sender,
            requestedOwner: address(0),
            consumers: consumers
        });

        emit SubscriptionCreated(currentSubId, msg.sender);
        return currentSubId;
    }

    function requestSubscriptionOwnerTransfer(
        uint64 subId,
        address newOwner
    ) external onlySubOwner(subId) nonReentrant {
        // Proposing to address(0) would never be claimable so don't need to check.
        if (s_subscriptionConfigs[subId].requestedOwner != newOwner) {
            s_subscriptionConfigs[subId].requestedOwner = newOwner;
            emit SubscriptionOwnerTransferRequested(subId, msg.sender, newOwner);
        }
    }

    function acceptSubscriptionOwnerTransfer(uint64 subId) external nonReentrant {
        if (s_subscriptionConfigs[subId].owner == address(0)) {
            revert InvalidSubscription();
        }
        if (s_subscriptionConfigs[subId].requestedOwner != msg.sender) {
            revert MustBeRequestedOwner(s_subscriptionConfigs[subId].requestedOwner);
        }
        address oldOwner = s_subscriptionConfigs[subId].owner;
        s_subscriptionConfigs[subId].owner = msg.sender;
        s_subscriptionConfigs[subId].requestedOwner = address(0);
        emit SubscriptionOwnerTransferred(subId, oldOwner, msg.sender);
    }

    function removeConsumer(
        uint64 subId,
        address consumer
    ) external onlySubOwner(subId) nonReentrant {
        if (s_consumers[consumer][subId] == 0) {
            revert InvalidConsumer(subId, consumer);
        }
        // Note bounded by MAX_CONSUMERS
        address[] memory consumers = s_subscriptionConfigs[subId].consumers;
        uint256 lastConsumerIndex = consumers.length - 1;
        for (uint256 i = 0; i < consumers.length; i++) {
            if (consumers[i] == consumer) {
                address last = consumers[lastConsumerIndex];
                // Storage write to preserve last element
                s_subscriptionConfigs[subId].consumers[i] = last;
                // Storage remove last element
                s_subscriptionConfigs[subId].consumers.pop();
                break;
            }
        }
        delete s_consumers[consumer][subId];
        emit SubscriptionConsumerRemoved(subId, consumer);
    }

    function addConsumer(uint64 subId, address consumer) external onlySubOwner(subId) nonReentrant {
        // Already maxed, cannot add any more consumers.
        if (s_subscriptionConfigs[subId].consumers.length >= MAX_CONSUMERS) {
            revert TooManyConsumers();
        }
        if (s_consumers[consumer][subId] != 0) {
            // Idempotence - do nothing if already added.
            // Ensures uniqueness in s_subscriptions[subId].consumers.
            return;
        }
        // Initialize the nonce to 1, indicating the consumer is allocated.
        s_consumers[consumer][subId] = 1;
        s_subscriptionConfigs[subId].consumers.push(consumer);

        emit SubscriptionConsumerAdded(subId, consumer);
    }

    function cancelSubscription(
        uint64 subId,
        address to
    ) external onlySubOwner(subId) nonReentrant {
        if (pendingRequestExists(subId)) {
            revert PendingRequestExists();
        }
        cancelSubscriptionHelper(subId, to);
    }

    function cancelSubscriptionHelper(uint64 subId, address to) private nonReentrant {
        SubscriptionConfig memory subConfig = s_subscriptionConfigs[subId];
        Subscription memory sub = s_subscriptions[subId];
        uint96 balance = sub.balance;
        // Note bounded by MAX_CONSUMERS;
        // If no consumers, does nothing.
        for (uint256 i = 0; i < subConfig.consumers.length; i++) {
            delete s_consumers[subConfig.consumers[i]][subId];
        }
        delete s_subscriptionConfigs[subId];
        delete s_subscriptions[subId];
        s_totalBalance -= balance;
        //fix this
        payable(address(to)).transfer(uint256(balance));
        emit SubscriptionCanceled(subId, to, balance);
    }

    function pendingRequestExists(uint64 subId) public view returns (bool) {

        // SubscriptionConfig memory subConfig = s_subscriptionConfigs[subId];
        // for (uint256 i = 0; i < subConfig.consumers.length; i++) {
        //     for (uint256 j = 0; j < s_provingKeyHashes.length; j++) {
        //         (uint256 reqId, ) = computeRequestId(
        //             s_provingKeyHashes[j],
        //             subConfig.consumers[i],
        //             subId,
        //             s_consumers[subConfig.consumers[i]][subId]
        //         );
        //         if (s_requestCommitments[reqId] != 0) {
        //             return true;
        //         }
        //     }
        // }
        return false;
    }

    modifier onlySubOwner(uint64 subId) {
        address owner = s_subscriptionConfigs[subId].owner;
        if (owner == address(0)) {
            revert InvalidSubscription();
        }
        if (msg.sender != owner) {
            revert MustBeSubOwner(owner);
        }
        _;
    }

    modifier nonReentrant() {
        // if (s_config.reentrancyLock) {
        //     revert Reentrant();
        // }
        _;
    }

    function typeAndVersion() external pure virtual returns (string memory) {
        return "Subscription 0.1";
    }
}
