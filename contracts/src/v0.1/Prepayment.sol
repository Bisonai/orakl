// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/VRFCoordinatorV2.sol

import "@openzeppelin/contracts/access/AccessControlEnumerable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/CoordinatorBaseInterface.sol";
import "./interfaces/PrepaymentInterface.sol";
import "./interfaces/TypeAndVersionInterface.sol";

contract Prepayment is
    AccessControlEnumerable,
    Ownable,
    PrepaymentInterface,
    TypeAndVersionInterface
{
    uint16 public constant MAX_CONSUMERS = 100;
    bytes32 public constant WITHDRAWER_ROLE = keccak256("WITHDRAWER_ROLE");
    bytes32 public constant ORACLE_ROLE = keccak256("ORACLE_ROLE");

    uint96 private s_totalBalance;

    uint64 private s_currentAccId;

    uint96 public s_withdrawable;

    /* consumer */ /* accId */ /* nonce */
    mapping(address => mapping(uint64 => uint64)) private s_consumers;

    /* accId */ /* AccountConfig */
    mapping(uint64 => AccountConfig) private s_accountConfigs;

    /* accId */ /* account */
    mapping(uint64 => Account) private s_accounts;

    struct Account {
        // There are only 1e9*1e18 = 1e27 juels in existence, so the balance can fit in uint96 (2^96 ~ 7e28)
        uint96 balance; // Common KLAY balance used for all consumer requests.
        uint64 reqCount; // For fee tiers
    }

    struct AccountConfig {
        address owner; // Owner can fund/withdraw/cancel the acc.
        address requestedOwner; // For safely transferring acc ownership.
        // Maintains the list of keys in s_consumers.
        // We do this for 2 reasons:
        // 1. To be able to clean up all keys from s_consumers when canceling an account.
        // 2. To be able to return the list of all consumers in getAccount.
        // Note that we need the s_consumers map to be able to directly check if a
        // consumer is valid without reading all the consumers from storage.
        address[] consumers;
    }

    CoordinatorBaseInterface[] private s_coordinators;

    error TooManyConsumers();
    error InsufficientBalance();
    error InsufficientConsumerBalance();
    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error MustBeAccountOwner(address owner);
    error PendingRequestExists();
    error MustBeRequestedOwner(address proposedOwner);
    error MustBeWithdrawer(address notWithdrawer);
    error MustBeOracle(address notOracle);

    event AccountCreated(uint64 indexed accId, address owner);
    event AccountFunded(uint64 indexed accId, uint256 oldBalance, uint256 newBalance);
    event AccountDecreased(uint64 indexed accId, uint256 oldBalance, uint256 newBalance);
    event AccountConsumerAdded(uint64 indexed accId, address consumer);
    event AccountConsumerRemoved(uint64 indexed accId, address consumer);
    event AccountCanceled(uint64 indexed accId, address to, uint256 amount);
    event AccountOwnerTransferRequested(uint64 indexed accId, address from, address to);
    event AccountOwnerTransferred(uint64 indexed accId, address from, address to);
    event FundsWithdrawn(address to, uint256 amount);

    modifier onlyAccOwner(uint64 accId) {
        address owner = s_accountConfigs[accId].owner;
        if (owner == address(0)) {
            revert InvalidAccount();
        }
        if (msg.sender != owner) {
            revert MustBeAccountOwner(owner);
        }
        _;
    }

    modifier onlyWithdrawer() {
        if (!hasRole(WITHDRAWER_ROLE, msg.sender)) {
            revert MustBeWithdrawer(msg.sender);
        }
        _;
    }

    modifier onlyOracle() {
        if (!hasRole(ORACLE_ROLE, msg.sender)) {
            revert MustBeOracle(msg.sender);
        }
        _;
    }

    constructor() {
        _setupRole(DEFAULT_ADMIN_ROLE, _msgSender());
    }

    receive() external payable {}

    function getTotalBalance() external view returns (uint256) {
        return s_totalBalance;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getAccount(
        uint64 accId
    )
        external
        view
        returns (uint96 balance, uint64 reqCount, address owner, address[] memory consumers)
    {
        if (s_accountConfigs[accId].owner == address(0)) {
            revert InvalidAccount();
        }
        return (
            s_accounts[accId].balance,
            s_accounts[accId].reqCount,
            s_accountConfigs[accId].owner,
            s_accountConfigs[accId].consumers
        );
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function createAccount() external returns (uint64) {
        s_currentAccId++;
        uint64 currentAccId = s_currentAccId;
        address[] memory consumers = new address[](0);
        s_accounts[currentAccId] = Account({balance: 0, reqCount: 0});
        s_accountConfigs[currentAccId] = AccountConfig({
            owner: msg.sender,
            requestedOwner: address(0),
            consumers: consumers
        });

        emit AccountCreated(currentAccId, msg.sender);
        return currentAccId;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function requestAccountOwnerTransfer(
        uint64 accId,
        address newOwner
    ) external onlyAccOwner(accId) {
        // Proposing to address(0) would never be claimable so don't need to check.
        if (s_accountConfigs[accId].requestedOwner != newOwner) {
            s_accountConfigs[accId].requestedOwner = newOwner;
            emit AccountOwnerTransferRequested(accId, msg.sender, newOwner);
        }
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function acceptAccountOwnerTransfer(uint64 accId) external override {
        if (s_accountConfigs[accId].owner == address(0)) {
            revert InvalidAccount();
        }
        if (s_accountConfigs[accId].requestedOwner != msg.sender) {
            revert MustBeRequestedOwner(s_accountConfigs[accId].requestedOwner);
        }
        address oldOwner = s_accountConfigs[accId].owner;
        s_accountConfigs[accId].owner = msg.sender;
        s_accountConfigs[accId].requestedOwner = address(0);
        emit AccountOwnerTransferred(accId, oldOwner, msg.sender);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function removeConsumer(uint64 accId, address consumer) external override onlyAccOwner(accId) {
        if (s_consumers[consumer][accId] == 0) {
            revert InvalidConsumer(accId, consumer);
        }
        // Note bounded by MAX_CONSUMERS
        address[] memory consumers = s_accountConfigs[accId].consumers;
        uint256 lastConsumerIndex = consumers.length - 1;
        for (uint256 i = 0; i < consumers.length; i++) {
            if (consumers[i] == consumer) {
                address last = consumers[lastConsumerIndex];
                // Storage write to preserve last element
                s_accountConfigs[accId].consumers[i] = last;
                // Storage remove last element
                s_accountConfigs[accId].consumers.pop();
                break;
            }
        }
        delete s_consumers[consumer][accId];
        emit AccountConsumerRemoved(accId, consumer);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function addConsumer(uint64 accId, address consumer) external override onlyAccOwner(accId) {
        // Already maxed, cannot add any more consumers.
        if (s_accountConfigs[accId].consumers.length >= MAX_CONSUMERS) {
            revert TooManyConsumers();
        }
        if (s_consumers[consumer][accId] != 0) {
            // Idempotence - do nothing if already added.
            // Ensures uniqueness in s_accounts[accId].consumers.
            return;
        }
        // Initialize the nonce to 1, indicating the consumer is allocated.
        s_consumers[consumer][accId] = 1;
        s_accountConfigs[accId].consumers.push(consumer);

        emit AccountConsumerAdded(accId, consumer);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function cancelAccount(uint64 accId, address to) external override onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }
        cancelAccountHelper(accId, to);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function deposit(uint64 accId) external payable override {
        if (msg.sender.balance < msg.value) {
            revert InsufficientConsumerBalance();
        }
        uint96 amount = uint96(msg.value); // FIXME
        uint96 oldBalance = s_accounts[accId].balance;
        s_accounts[accId].balance += amount;
        s_totalBalance += amount;
        emit AccountFunded(accId, oldBalance, oldBalance + amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function withdraw(uint64 accId, uint96 amount) external override onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        uint256 oldBalance = s_accounts[accId].balance;
        if ((oldBalance < amount) || (address(this).balance < amount)) {
            revert InsufficientBalance();
        }

        s_accounts[accId].balance -= amount;

        (bool sent, ) = msg.sender.call{value: amount}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit AccountDecreased(accId, oldBalance, oldBalance - amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function ownerWithdraw(uint96 amount) external onlyWithdrawer {
        if (address(this).balance < amount) {
            revert InsufficientBalance();
        }

        s_withdrawable -= amount;

        (bool sent, ) = msg.sender.call{value: amount}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit FundsWithdrawn(msg.sender, amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function chargeFee(uint64 accId, uint96 amount) external override onlyOracle {
        uint96 oldBalance = s_accounts[accId].balance;
        if (oldBalance < amount) {
            revert InsufficientBalance();
        }

        s_accounts[accId].balance -= amount;
        s_accounts[accId].reqCount += 1;
        s_withdrawable += amount;

        emit AccountDecreased(accId, oldBalance, oldBalance - amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getNonce(address consumer, uint64 accId) external view override returns (uint64) {
        return s_consumers[consumer][accId];
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function increaseNonce(address consumer, uint64 accId) external returns (uint64) {
        uint64 currentNonce = s_consumers[consumer][accId];
        uint64 nonce = currentNonce + 1;
        s_consumers[consumer][accId] = nonce;
        return nonce;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getAccountOwner(uint64 accId) external view returns (address owner) {
        return s_accountConfigs[accId].owner;
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Prepayment v0.1";
    }

    /**
     * @inheritdoc PrepaymentInterface
     * @dev Looping is bounded to MAX_CONSUMERS*(number of keyhashes).
     * @dev Used to disable subscription canceling while outstanding request are present.
     */
    function pendingRequestExists(uint64 accId) public view override returns (bool) {
        AccountConfig memory accConfig = s_accountConfigs[accId];
        for (uint256 i = 0; i < accConfig.consumers.length; i++) {
            for (uint256 j = 0; j < s_coordinators.length; j++) {
                if (
                    s_coordinators[j].pendingRequestExists(
                        accId,
                        accConfig.consumers[i],
                        s_consumers[accConfig.consumers[i]][accId]
                    )
                ) {
                    return true;
                }
            }
        }
        return false;
    }

    function addCoordinator(CoordinatorBaseInterface coordinator) public onlyOwner {
        s_coordinators.push(coordinator);
    }

    function removeCoordinator(CoordinatorBaseInterface coordinator) public onlyOwner {
        uint256 lastCoordinatorIndex = s_coordinators.length - 1;
        for (uint256 i = 0; i < s_coordinators.length; i++) {
            if (s_coordinators[i] == coordinator) {
                CoordinatorBaseInterface last = s_coordinators[lastCoordinatorIndex];
                // Storage write to preserve last element
                s_coordinators[i] = last;
                // Storage remove last element
                s_coordinators.pop();
                break;
            }
        }
    }

    function cancelAccountHelper(uint64 accId, address to) private {
        AccountConfig memory accConfig = s_accountConfigs[accId];
        Account memory acc = s_accounts[accId];
        uint96 balance = acc.balance;

        // Note bounded by MAX_CONSUMERS;
        // If no consumers, does nothing.
        for (uint256 i = 0; i < accConfig.consumers.length; i++) {
            delete s_consumers[accConfig.consumers[i]][accId];
        }

        delete s_accountConfigs[accId];
        delete s_accounts[accId];
        s_totalBalance -= balance;

        (bool sent, ) = to.call{value: balance}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit AccountCanceled(accId, to, balance);
    }
}
