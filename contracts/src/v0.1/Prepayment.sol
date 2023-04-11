// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/VRFCoordinatorV2.sol

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/CoordinatorBaseInterface.sol";
import "./interfaces/PrepaymentInterface.sol";
import "./interfaces/TypeAndVersionInterface.sol";

contract Prepayment is Ownable, PrepaymentInterface, TypeAndVersionInterface {
    uint16 public constant MAX_CONSUMERS = 100;
    uint8 public constant MIN_BURN_RATIO = 0;
    uint8 public constant MAX_BURN_RATIO = 100;

    uint256 private sTotalBalance;

    uint64 private sCurrentAccId;
    uint8 public sBurnRatio = 20; //20%

    /* consumer */
    /* accId */
    /* nonce */
    mapping(address => mapping(uint64 => uint64)) private sConsumers;

    /* accId */
    /* AccountConfig */
    mapping(uint64 => AccountConfig) private sAccountConfigs;

    /* accId */
    /* account */
    mapping(uint64 => Account) private sAccounts;

    mapping(address => uint256) public sNodes;

    struct Account {
        // There are only 1e9*1e18 = 1e27 juels in existence, so the balance can fit in uint256 (2^96 ~ 7e28)
        uint256 balance; // Common KLAY balance used for all consumer requests.
        uint64 reqCount; // For fee tiers
        string accType;
    }

    struct AccountConfig {
        address owner; // Owner can fund/withdraw/cancel the acc.
        address requestedOwner; // For safely transferring acc ownership.
        // Maintains the list of keys in sConsumers.
        // We do this for 2 reasons:
        // 1. To be able to clean up all keys from sConsumers when canceling an account.
        // 2. To be able to return the list of all consumers in getAccount.
        // Note that we need the sConsumers map to be able to directly check if a
        // consumer is valid without reading all the consumers from storage.
        address[] consumers;
    }

    CoordinatorBaseInterface[] public sCoordinators;
    mapping(address => bool) private sIsCoordinators;

    error TooManyConsumers();
    error InsufficientBalance();
    error InvalidConsumer(uint64 accId, address consumer);
    error InvalidAccount();
    error MustBeAccountOwner(address owner);
    error PendingRequestExists();
    error MustBeRequestedOwner(address proposedOwner);
    error ZeroAmount();
    error CoordinatorExists();
    error InvalidBurnRatio();
    error BurnFeeFailed();
    error InvalidCoordinator();

    event AccountCreated(uint64 indexed accId, address owner, string accType);
    event AccountCanceled(uint64 indexed accId, address to, uint256 amount);
    event AccountBalanceIncreased(uint64 indexed accId, uint256 oldBalance, uint256 newBalance);
    event AccountBalanceDecreased(
        uint64 indexed accId,
        uint256 oldBalance,
        uint256 newBalance,
        uint256 burnAmount
    );
    event AccountConsumerAdded(uint64 indexed accId, address consumer);
    event AccountConsumerRemoved(uint64 indexed accId, address consumer);
    event AccountOwnerTransferRequested(uint64 indexed accId, address from, address to);
    event AccountOwnerTransferred(uint64 indexed accId, address from, address to);
    event FundsWithdrawn(address to, uint256 amount);
    event BurnRatioSet(uint16 ratio);

    modifier onlyAccOwner(uint64 accId) {
        address owner = sAccountConfigs[accId].owner;
        if (owner == address(0)) {
            revert InvalidAccount();
        }
        if (msg.sender != owner) {
            revert MustBeAccountOwner(owner);
        }
        _;
    }

    modifier onlyCoordinator() {
        bool isCoordinator = false;
        for (uint256 i = 0; i < sCoordinators.length; i++) {
            if (sCoordinators[i] == CoordinatorBaseInterface(msg.sender)) {
                isCoordinator = true;
                break;
            }
        }
        if (isCoordinator == false) {
            revert InvalidCoordinator();
        }
        _;
    }

    /**
     * The function allows to update a "burn ratio" that represents a
     * partial amount of payment for a service that will be burnt.
     */
    function setBurnRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_BURN_RATIO || ratio > MAX_BURN_RATIO) {
            revert InvalidBurnRatio();
        }
        sBurnRatio = ratio;
        emit BurnRatioSet(ratio);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getTotalBalance() external view returns (uint256) {
        return sTotalBalance;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getAccount(
        uint64 accId
    )
        external
        view
        returns (
            uint256 balance,
            uint64 reqCount,
            string memory accType,
            address owner,
            address[] memory consumers
        )
    {
        if (sAccountConfigs[accId].owner == address(0)) {
            revert InvalidAccount();
        }
        return (
            sAccounts[accId].balance,
            sAccounts[accId].reqCount,
            sAccounts[accId].accType,
            sAccountConfigs[accId].owner,
            sAccountConfigs[accId].consumers
        );
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function createAccount() external returns (uint64) {
        sCurrentAccId++;
        uint64 currentAccId = sCurrentAccId;
        address[] memory consumers = new address[](0);
        string memory accType = "reg";
        if (sIsCoordinators[msg.sender]) {
            accType = "tmp";
        }
        sAccounts[currentAccId] = Account({balance: 0, reqCount: 0, accType: accType});
        sAccountConfigs[currentAccId] = AccountConfig({
            owner: msg.sender,
            requestedOwner: address(0),
            consumers: consumers
        });

        emit AccountCreated(currentAccId, msg.sender, accType);
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
        if (sAccountConfigs[accId].requestedOwner != newOwner) {
            sAccountConfigs[accId].requestedOwner = newOwner;
            emit AccountOwnerTransferRequested(accId, msg.sender, newOwner);
        }
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function acceptAccountOwnerTransfer(uint64 accId) external {
        if (sAccountConfigs[accId].owner == address(0)) {
            revert InvalidAccount();
        }
        if (sAccountConfigs[accId].requestedOwner != msg.sender) {
            revert MustBeRequestedOwner(sAccountConfigs[accId].requestedOwner);
        }
        address oldOwner = sAccountConfigs[accId].owner;
        sAccountConfigs[accId].owner = msg.sender;
        sAccountConfigs[accId].requestedOwner = address(0);
        emit AccountOwnerTransferred(accId, oldOwner, msg.sender);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function removeConsumer(uint64 accId, address consumer) external onlyAccOwner(accId) {
        if (sConsumers[consumer][accId] == 0) {
            revert InvalidConsumer(accId, consumer);
        }
        // Note bounded by MAX_CONSUMERS
        address[] memory consumers = sAccountConfigs[accId].consumers;
        uint256 lastConsumerIndex = consumers.length - 1;
        for (uint256 i = 0; i < consumers.length; i++) {
            if (consumers[i] == consumer) {
                address last = consumers[lastConsumerIndex];
                // Storage write to preserve last element
                sAccountConfigs[accId].consumers[i] = last;
                // Storage remove last element
                sAccountConfigs[accId].consumers.pop();
                break;
            }
        }
        delete sConsumers[consumer][accId];
        emit AccountConsumerRemoved(accId, consumer);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function addConsumer(uint64 accId, address consumer) external onlyAccOwner(accId) {
        // Already maxed, cannot add any more consumers.
        if (sAccountConfigs[accId].consumers.length >= MAX_CONSUMERS) {
            revert TooManyConsumers();
        }
        if (sConsumers[consumer][accId] != 0) {
            // Idempotence - do nothing if already added.
            // Ensures uniqueness in sAccounts[accId].consumers.
            return;
        }
        // Initialize the nonce to 1, indicating the consumer is allocated.
        sConsumers[consumer][accId] = 1;
        sAccountConfigs[accId].consumers.push(consumer);

        emit AccountConsumerAdded(accId, consumer);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function cancelAccount(uint64 accId, address to) external onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }
        cancelAccountHelper(accId, to);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function deposit(uint64 accId) external payable {
        uint256 amount = msg.value;
        uint256 oldBalance = sAccounts[accId].balance;
        sAccounts[accId].balance += amount;
        sTotalBalance += amount;
        emit AccountBalanceIncreased(accId, oldBalance, oldBalance + amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function withdraw(uint64 accId, uint256 amount) external onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        uint256 oldBalance = sAccounts[accId].balance;
        if ((oldBalance < amount) || (address(this).balance < amount)) {
            revert InsufficientBalance();
        }

        sAccounts[accId].balance -= amount;

        (bool sent, ) = msg.sender.call{value: amount}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit AccountBalanceDecreased(accId, oldBalance, oldBalance - amount, 0);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function nodeWithdraw(uint256 amount) external {
        if (amount == 0) {
            revert ZeroAmount();
        }
        if (address(this).balance < amount) {
            revert InsufficientBalance();
        }
        uint256 withdrawable = sNodes[msg.sender];
        if (withdrawable < amount) {
            revert InsufficientBalance();
        }
        sNodes[msg.sender] -= amount;
        (bool sent, ) = msg.sender.call{value: amount}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit FundsWithdrawn(msg.sender, amount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function chargeFee(uint64 accId, uint256 amount, address node) external onlyCoordinator {
        uint256 oldBalance = sAccounts[accId].balance;
        if (oldBalance < amount) {
            revert InsufficientBalance();
        }

        sAccounts[accId].balance -= amount;
        sAccounts[accId].reqCount += 1;
        uint256 burnAmount = (amount * sBurnRatio) / 100;
        sNodes[node] += amount - burnAmount;
        if (burnAmount > 0) {
            (bool sent, ) = address(0).call{value: burnAmount}("");
            if (!sent) {
                revert BurnFeeFailed();
            }
        }

        emit AccountBalanceDecreased(accId, oldBalance, oldBalance - amount, burnAmount);
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getNonce(address consumer, uint64 accId) external view returns (uint64) {
        return sConsumers[consumer][accId];
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function increaseNonce(
        address consumer,
        uint64 accId
    ) external onlyCoordinator returns (uint64) {
        uint64 currentNonce = sConsumers[consumer][accId];
        uint64 nonce = currentNonce + 1;
        sConsumers[consumer][accId] = nonce;
        return nonce;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function getAccountOwner(uint64 accId) external view returns (address owner) {
        return sAccountConfigs[accId].owner;
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
    function pendingRequestExists(uint64 accId) public view returns (bool) {
        AccountConfig memory accConfig = sAccountConfigs[accId];
        for (uint256 i = 0; i < accConfig.consumers.length; i++) {
            for (uint256 j = 0; j < sCoordinators.length; j++) {
                if (
                    sCoordinators[j].pendingRequestExists(
                        accConfig.consumers[i],
                        accId,
                        sConsumers[accConfig.consumers[i]][accId]
                    )
                ) {
                    return true;
                }
            }
        }
        return false;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function addCoordinator(address coordinator) public onlyOwner {
        if (sIsCoordinators[coordinator]) {
            revert CoordinatorExists();
        }
        sCoordinators.push(CoordinatorBaseInterface(coordinator));
        sIsCoordinators[coordinator] = true;
    }

    /**
     * @inheritdoc PrepaymentInterface
     */
    function removeCoordinator(address coordinator) public onlyOwner {
        for (uint256 i = 0; i < sCoordinators.length; i++) {
            if (sCoordinators[i] == CoordinatorBaseInterface(coordinator)) {
                CoordinatorBaseInterface last = sCoordinators[sCoordinators.length - 1];
                sCoordinators[i] = last;
                sCoordinators.pop();
                break;
            }
        }
        delete sIsCoordinators[coordinator];
    }

    /*
     * @notice Remove consumers and account related information.
     * @notice Return remaining balance.
     * @param accId - ID of the account
     * @param to - Where to send the remaining KLAY to
     */
    function cancelAccountHelper(uint64 accId, address to) private {
        AccountConfig memory accConfig = sAccountConfigs[accId];
        Account memory acc = sAccounts[accId];
        uint256 balance = acc.balance;

        // Note bounded by MAX_CONSUMERS;
        // If no consumers, does nothing.
        for (uint256 i = 0; i < accConfig.consumers.length; i++) {
            delete sConsumers[accConfig.consumers[i]][accId];
        }

        delete sAccountConfigs[accId];
        delete sAccounts[accId];
        sTotalBalance -= balance;

        (bool sent, ) = to.call{value: balance}("");
        if (!sent) {
            revert InsufficientBalance();
        }

        emit AccountCanceled(accId, to, balance);
    }
}
