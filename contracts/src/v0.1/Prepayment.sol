// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./Account.sol";
import "./interfaces/IAccount.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";

contract Prepayment is Ownable, IPrepayment, ITypeAndVersion {
    uint8 public constant MIN_RATIO = 0;
    uint8 public constant MAX_RATIO = 100;
    uint8 private sBurnFeeRatio = 20; // %
    uint8 private sProtocolFeeRatio = 5; // %

    address private sProtocolFeeRecipient;

    // Coordinator
    ICoordinatorBase[] public sCoordinators;

    /* coordinator address */
    /* association */
    mapping(address => bool) private sIsCoordinator;

    /* temporary account ID */
    /* nonce */
    mapping(uint64 => uint64) private sTemporaryNonce;

    // Account
    uint64 private sCurrentAccId;
    uint64 private sCurrentTmpAccId;

    /* accId */
    /* Account */
    mapping(uint64 => Account) private sAccIdToAccount;

    mapping(uint64 => bool) sIsTemporaryAccount;

    struct TemporaryAccountConfig {
        uint256 balance;
        uint64 reqCount;
        address owner;
        /* uint64 nonce; */
        address[] consumer;
    }

    /* accID */
    /* TemporaryAccountConfig */
    mapping(uint64 => TemporaryAccountConfig) private sAccIdToTmpAccConfig;

    error PendingRequestExists();
    error InvalidCoordinator();
    error InvalidAccount();
    error MustBeAccountOwner();
    error RatioOutOfBounds();
    error FailedToDeposit();
    error FailedToWithdraw();
    error CoordinatorExists();
    error InsufficientBalance();
    error BurnFeeFailed();
    error OperatorFeeFailed();
    error ProtocolFeeFailed();

    event AccountCreated(uint64 indexed accId, address account, address owner);
    event TemporaryAccountCreated(uint64 indexed accId, address owner);
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
    event BurnRatioSet(uint8 ratio);
    event ProtocolFeeRatioSet(uint8 ratio);
    event CoordinatorAdded(address coordinator);
    event CoordinatorRemoved(address coordinator);
    event AccountOwnerTransferRequested(uint64 indexed accId, address from, address to);
    event AccountOwnerTransferred(uint64 indexed accId, address from, address to);

    modifier onlyAccountOwner(uint64 accId) {
        if (sAccIdToAccount[accId].getOwner() != msg.sender) {
            revert MustBeAccountOwner();
        }
        _;
    }

    modifier onlyCoordinator() {
        if (!sIsCoordinator[msg.sender]) {
            revert InvalidCoordinator();
        }
        _;
    }

    constructor(address protocolFeeRecipient) {
        sProtocolFeeRecipient = protocolFeeRecipient;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function isValidAccount(uint64 accId) external view returns (bool) {
        Account account = sAccIdToAccount[accId];
        if (address(account) != address(0) || sIsTemporaryAccount[accId]) {
            return true;
        } else {
            revert InvalidAccount();
        }
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getBalance(uint64 accId) external view returns (uint256 balance) {
        Account account = sAccIdToAccount[accId];
        return account.getBalance();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getReqCount(uint64 accId) external view returns (uint64) {
        return sAccIdToAccount[accId].getReqCount();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccount(
        uint64 accId
    )
        external
        view
        returns (uint256 balance, uint64 reqCount, address owner, address[] memory consumers)
    {
        Account account = sAccIdToAccount[accId];

        if (address(account) != address(0)) {
            // regular account
            return account.getAccount();
        } else if (sIsTemporaryAccount[accId]) {
            // temporary account
            TemporaryAccountConfig memory tmpAccConfig = sAccIdToTmpAccConfig[accId];
            return (
                tmpAccConfig.balance,
                tmpAccConfig.reqCount,
                tmpAccConfig.owner,
                tmpAccConfig.consumer
            );
        } else {
            revert InvalidAccount();
        }
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccountOwner(uint64 accId) external view returns (address) {
        Account account = sAccIdToAccount[accId];
        return account.getOwner();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getCoordinators() external view returns (ICoordinatorBase[] memory) {
        return sCoordinators;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getNonce(uint64 accId, address consumer) external view returns (uint64) {
        Account account = sAccIdToAccount[accId];
        if (address(account) != address(0)) {
            return sAccIdToAccount[accId].getNonce(consumer);
        } else {
            // Temporary account has nonce always equal to 1
            // FIXME should we define mapping for consumer?
            return sTemporaryNonce[accId];
        }
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getProtocolFeeRecipient() external view returns (address) {
        return sProtocolFeeRecipient;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function setProtocolFeeRecipient(address protocolFeeRecipient) external onlyOwner {
        sProtocolFeeRecipient = protocolFeeRecipient;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createAccount() external returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;

        Account acc = new Account(currentAccId, msg.sender);
        sAccIdToAccount[currentAccId] = acc;

        emit AccountCreated(currentAccId, address(acc), msg.sender);
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createTemporaryAccount() external returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;

        sIsTemporaryAccount[currentAccId] = true;
        sTemporaryNonce[currentAccId] = 1;

        emit TemporaryAccountCreated(currentAccId, msg.sender);
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function requestAccountOwnerTransfer(
        uint64 accId,
        address requestedOwner
    ) external onlyAccountOwner(accId) {
        Account account = sAccIdToAccount[accId];
        account.requestAccountOwnerTransfer(requestedOwner);
        emit AccountOwnerTransferRequested(accId, msg.sender, requestedOwner);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function acceptAccountOwnerTransfer(uint64 accId) external {
        Account account = sAccIdToAccount[accId];
        address newOwner = msg.sender;
        address oldOwner = account.getOwner();
        account.acceptAccountOwnerTransfer(newOwner);
        emit AccountOwnerTransferred(accId, oldOwner, newOwner);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function cancelAccount(uint64 accId, address to) external onlyAccountOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        Account account = sAccIdToAccount[accId];
        uint256 balance = account.getBalance();

        account.cancelAccount(to);
        delete sAccIdToAccount[accId];

        emit AccountCanceled(accId, to, balance);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function addConsumer(uint64 accId, address consumer) external onlyAccountOwner(accId) {
        sAccIdToAccount[accId].addConsumer(consumer);
        emit AccountConsumerAdded(accId, consumer);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function removeConsumer(uint64 accId, address consumer) external onlyAccountOwner(accId) {
        sAccIdToAccount[accId].removeConsumer(consumer);
        emit AccountConsumerRemoved(accId, consumer);
    }

    /**
     * @notice Return the current burn ratio that represents the
     * @notice percentage of $KLAY fee that is burnt during fulfillment
     * @notice of every request.
     */
    function getBurnFeeRatio() external view returns (uint8) {
        return sBurnFeeRatio;
    }

    /**
     * @notice The function allows to update a "burn ratio" that represents a
     * @notice partial amount of payment for the Orakl Network service that
     * @notice will be burnt.
     *
     * @param ratio in a range 0 - 100 % of a fee to be burnt
     */
    function setBurnFeeRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_RATIO || ratio > MAX_RATIO) {
            revert RatioOutOfBounds();
        }
        sBurnFeeRatio = ratio;
        emit BurnRatioSet(ratio);
    }

    /**
     * @notice Return the current protocol fee ratio that represents
     * @notice the percentage of $KLAY fee that is charged for every
     * @notice finalizes fulfillment.
     */
    function getProtocolFeeRatio() external view returns (uint8) {
        return sProtocolFeeRatio;
    }

    /**
     * @notice The function allows to update a protocol fee.
     *
     * @param ratio in a range 0 - 100 % of a fee to be burnt
     */
    function setProtocolFeeRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_RATIO || ratio > MAX_RATIO) {
            revert RatioOutOfBounds();
        }
        sProtocolFeeRatio = ratio;
        emit ProtocolFeeRatioSet(ratio);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function deposit(uint64 accId) external payable {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) {
            revert InvalidAccount();
        }
        uint256 amount = msg.value;
        uint256 balance = account.getBalance();

        (bool sent, ) = payable(account).call{value: msg.value}("");
        if (!sent) {
            revert FailedToDeposit();
        }

        emit AccountBalanceIncreased(accId, balance, balance + amount);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function depositTemporary(uint64 accId) external payable {
        emit AccountBalanceIncreased(accId, 0, msg.value);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function withdraw(uint64 accId, uint256 amount) external onlyAccountOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        Account account = sAccIdToAccount[accId];
        (bool sent, uint256 balance) = account.withdraw(amount);
        if (!sent) {
            revert FailedToWithdraw();
        }

        emit AccountBalanceDecreased(accId, balance + amount, balance, 0);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function chargeFee(
        uint64 accId,
        uint256 amount,
        address operatorFeeRecipient
    ) external onlyCoordinator {
        Account account = sAccIdToAccount[accId];
        uint256 balance = account.getBalance();

        if (balance < amount) {
            revert InsufficientBalance();
        }

        uint256 burnFee = (amount * sBurnFeeRatio) / 100;
        uint256 protocolFee = (amount * sProtocolFeeRatio) / 100;
        uint256 operatorFee = amount - burnFee - protocolFee;

        account.chargeFee(
            burnFee,
            operatorFee,
            operatorFeeRecipient,
            protocolFee,
            sProtocolFeeRecipient
        );

        emit AccountBalanceDecreased(accId, balance, balance - amount, burnFee);
    }

    function chargeFee(
        uint64 accId,
        address operatorFeeRecipient
    ) external onlyCoordinator returns (uint256) {
        TemporaryAccountConfig memory tmpAccConfig = sAccIdToTmpAccConfig[accId];
        uint256 amount = tmpAccConfig.balance;
        sAccIdToTmpAccConfig[accId].balance = 0;

        uint256 burnFee = (amount * sBurnFeeRatio) / 100;
        uint256 protocolFee = (amount * sProtocolFeeRatio) / 100;
        uint256 operatorFee = amount - burnFee - protocolFee;

        if (burnFee > 0) {
            (bool sent, ) = address(0).call{value: burnFee}("");
            if (!sent) {
                revert BurnFeeFailed();
            }
        }

        if (operatorFee > 0) {
            (bool sent, ) = operatorFeeRecipient.call{value: operatorFee}("");
            if (!sent) {
                revert OperatorFeeFailed();
            }
        }

        if (protocolFee > 0) {
            (bool sent, ) = sProtocolFeeRecipient.call{value: protocolFee}("");
            if (!sent) {
                revert ProtocolFeeFailed();
            }
        }

        emit AccountBalanceDecreased(accId, amount, 0, burnFee);

        return amount;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function increaseNonce(
        uint64 accId,
        address consumer
    ) external onlyCoordinator returns (uint64) {
        Account account = sAccIdToAccount[accId];
        if (address(account) != address(0)) {
            return account.increaseNonce(consumer);
        } else {
            // FIXME should we define mapping for consumer?
            uint64 nonce = sTemporaryNonce[accId] + 1;
            sTemporaryNonce[accId] = nonce;
            return nonce;
        }
    }

    /**
     * @inheritdoc IPrepayment
     */
    function addCoordinator(address coordinator) public onlyOwner {
        if (sIsCoordinator[coordinator]) {
            revert CoordinatorExists();
        }
        sCoordinators.push(ICoordinatorBase(coordinator));
        sIsCoordinator[coordinator] = true;

        emit CoordinatorAdded(coordinator);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function removeCoordinator(address coordinator) public onlyOwner {
        if (!sIsCoordinator[coordinator]) {
            revert InvalidCoordinator();
        }

        uint256 coordinatorsLength = sCoordinators.length;
        for (uint256 i; i < coordinatorsLength; ++i) {
            if (sCoordinators[i] == ICoordinatorBase(coordinator)) {
                ICoordinatorBase last = sCoordinators[coordinatorsLength - 1];
                sCoordinators[i] = last;
                sCoordinators.pop();
                break;
            }
        }

        delete sIsCoordinator[coordinator];
        emit CoordinatorRemoved(coordinator);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Prepayment v0.1";
    }

    /**
     * @inheritdoc IPrepayment
     * @dev Looping is bounded to MAX_CONSUMERS*(number of keyhashes).
     * @dev Use to reject account cancellation while outstanding request are present.
     */
    function pendingRequestExists(uint64 accId) public view returns (bool) {
        Account account = sAccIdToAccount[accId];
        address[] memory consumers = account.getConsumers();
        uint256 consumersLength = consumers.length;
        uint256 coordinatorsLength = sCoordinators.length;

        for (uint256 i = 0; i < consumersLength; i++) {
            address consumer = consumers[i];
            uint64 nonce = account.getNonce(consumer);
            for (uint256 j = 0; j < coordinatorsLength; j++) {
                if (sCoordinators[j].pendingRequestExists(consumer, accId, nonce)) {
                    return true;
                }
            }
        }
        return false;
    }
}
