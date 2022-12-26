// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "openzeppelin-solidity/contracts/access/AccessControlEnumerable.sol";
import "openzeppelin-solidity/contracts/access/Ownable.sol";
import "./interfaces/PrepaymentInterface.sol";

contract Prepayment is Ownable, AccessControlEnumerable, PrepaymentInterface {
    uint16 public constant MAX_CONSUMERS = 100;
    uint96 private s_totalBalance;
    error TooManyConsumers();
    error InsufficientBalance();
    error InvalidConsumer(uint64 subId, address consumer);
    error InvalidSubscription();
    error MustBeSubOwner(address owner);
    error PendingRequestExists();
    error MustBeRequestedOwner(address proposedOwner);
    error Reentrant();
      error ProvingKeyAlreadyRegistered(bytes32 keyHash);
  error NoSuchProvingKey(bytes32 keyHash);

    event SubscriptionCreated(uint64 indexed subId, address owner);
    event SubscriptionFunded(uint64 indexed subId, uint256 oldBalance, uint256 newBalance);
    event SubscriptionBanlanceDecreased(
        uint64 indexed subId,
        uint256 oldBalance,
        uint256 newBalance
    );
    event SubscriptionConsumerAdded(uint64 indexed subId, address consumer);
    event SubscriptionConsumerRemoved(uint64 indexed subId, address consumer);
    event SubscriptionCanceled(uint64 indexed subId, address to, uint256 amount);
    event SubscriptionOwnerTransferRequested(uint64 indexed subId, address from, address to);
    event SubscriptionOwnerTransferred(uint64 indexed subId, address from, address to);
  event ProvingKeyRegistered(bytes32 keyHash, address indexed oracle);
  event ProvingKeyDeregistered(bytes32 keyHash, address indexed oracle);
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
  /* keyHash */ /* oracle */
    mapping(bytes32 => address) private s_provingKeys;
    bytes32[] private s_provingKeyHashes;
     mapping(uint256 => bytes32) private s_requestCommitments;

    mapping(address => mapping(uint64 => uint64)) private s_consumers;

    /* subId */
    /* subscriptionConfig */
    mapping(uint64 => SubscriptionConfig) private s_subscriptionConfigs;

    /* subId */
    /* subscription */
    mapping(uint64 => Subscription) private s_subscriptions;

    // We make the sub count public so that its possible to
    // get all the current subscriptions via getSubscription.
    uint64 private s_currentSubId;

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

    function createSubscription() external returns (uint64) {
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

    // @Kelvin: removed nonReentrant. Can be done in reporter to cancel all pending fulfillment
    function requestSubscriptionOwnerTransfer(
        uint64 subId,
        address newOwner
    ) external onlySubOwner(subId) noPendingRequest {
        // Proposing to address(0) would never be claimable so don't need to check.
        if (s_subscriptionConfigs[subId].requestedOwner != newOwner) {
            s_subscriptionConfigs[subId].requestedOwner = newOwner;
            emit SubscriptionOwnerTransferRequested(subId, msg.sender, newOwner);
        }
    }

    function acceptSubscriptionOwnerTransfer(uint64 subId) external override noPendingRequest {
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
    ) external onlySubOwner(subId) noPendingRequest {
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

    function addConsumer(uint64 subId, address consumer) external onlySubOwner(subId) {
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
    ) external onlySubOwner(subId) noPendingRequest {
        if (pendingRequestExists(subId)) {
            revert PendingRequestExists();
        }
        cancelSubscriptionHelper(subId, to);
    }

    function cancelSubscriptionHelper(uint64 subId, address to) private {
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
    
  function hashOfKey(uint256[2] memory publicKey) public pure returns (bytes32) {
    return keccak256(abi.encode(publicKey));
  }
    function registerProvingKey(
        address oracle,
        uint256[2] calldata publicProvingKey
    ) external onlyOwner {
        bytes32 kh = hashOfKey(publicProvingKey);
        if (s_provingKeys[kh] != address(0)) {
            revert ProvingKeyAlreadyRegistered(kh);
        }
        s_provingKeys[kh] = oracle;
        s_provingKeyHashes.push(kh);
        emit ProvingKeyRegistered(kh, oracle);
    }

    /**
     * @notice Deregisters a proving key to an oracle.
     * @param publicProvingKey key that oracle can use to submit vrf fulfillments
     */
    function deregisterProvingKey(uint256[2] calldata publicProvingKey) external onlyOwner {
        bytes32 kh = hashOfKey(publicProvingKey);
        address oracle = s_provingKeys[kh];
        if (oracle == address(0)) {
            revert NoSuchProvingKey(kh);
        }
        delete s_provingKeys[kh];
        for (uint256 i = 0; i < s_provingKeyHashes.length; i++) {
            if (s_provingKeyHashes[i] == kh) {
                bytes32 last = s_provingKeyHashes[s_provingKeyHashes.length - 1];
                // Copy last element and overwrite kh to be deleted with it
                s_provingKeyHashes[i] = last;
                s_provingKeyHashes.pop();
            }
        }
        emit ProvingKeyDeregistered(kh, oracle);
    }

    function computeRequestId(
        bytes32 keyHash,
        address sender,
        uint64 subId,
        uint64 nonce
    ) private pure returns (uint256, uint256) {
        uint256 preSeed = uint256(keccak256(abi.encode(keyHash, sender, subId, nonce)));
        return (uint256(keccak256(abi.encode(keyHash, preSeed))), preSeed);
    }

    function pendingRequestExists(uint64 subId) public view returns (bool) {
        SubscriptionConfig memory subConfig = s_subscriptionConfigs[subId];
        for (uint256 i = 0; i < subConfig.consumers.length; i++) {
            for (uint256 j = 0; j < s_provingKeyHashes.length; j++) {
                (uint256 reqId, ) = computeRequestId(
                    s_provingKeyHashes[j],
                    subConfig.consumers[i],
                    subId,
                    s_consumers[subConfig.consumers[i]][subId]
                );
                if (s_requestCommitments[reqId] != 0) {
                    return true;
                }
            }
        }
        return false;
    }

    function decreaseSubBalance(uint64 subId, uint96 amount) external {
        if (s_subscriptions[subId].balance < amount) {
            revert InsufficientBalance();
        }
        s_subscriptions[subId].balance -= amount;
        //increase request count
        s_subscriptions[subId].reqCount += 1;
    }

    function deposit(uint64 subId) external payable {
        uint96 amount = uint96(msg.value);
        require(msg.sender.balance >= msg.value, "Insufficient account balance");
        uint256 oldBalance = s_subscriptions[subId].balance;
        s_subscriptions[subId].balance += amount;
        uint256 newBalance = s_subscriptions[subId].balance;
        s_totalBalance += amount;
        emit SubscriptionFunded(subId, oldBalance, newBalance);
    }

    function withdraw(uint64 subId, uint96 amount) external {
        if (s_subscriptions[subId].balance < amount) {
            revert InsufficientBalance();
        }
        require(address(this).balance >= amount, "Prepayment: Insufficient account balance");
        uint256 oldBalance = s_subscriptions[subId].balance;
        s_subscriptions[subId].balance -= amount;
        payable(msg.sender).transfer(amount);
        uint256 newBalance = s_subscriptions[subId].balance;
        emit SubscriptionBanlanceDecreased(subId, oldBalance, newBalance);
    }

    receive() external payable {}

    function getNonce(address consumer, uint64 subId) external view returns (uint64) {
        return s_consumers[consumer][subId];
    }

    function increaseNonce(address consumer, uint64 subId) external {
        s_consumers[consumer][subId] += 1;
    }

    function getSubOwner(uint64 subId) external view returns (address owner) {
        return s_subscriptionConfigs[subId].owner;
    }

    modifier noPendingRequest(uint64 subId){
        require(!pendingRequestExists(subId));
        _;
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

    modifier onlyWithdrawer() {
        require(
            owner() == msg.sender || hasRole(WITHDRAWER_ROLE, msg.sender),
            "Caller is not a withdrawer"
        );
        _;
    }

    function typeAndVersion() external pure virtual returns (string memory) {
        return "Subscription 0.1";
    }
}
