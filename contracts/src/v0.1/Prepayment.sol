// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./Account.sol";
import "./interfaces/IAccount.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";

/// @title Orakl Network Prepayment
/// @author Bisonai
/// @notice Prepayment is a type of prepaid payment solution which
/// @notice allows to controls two types of accounts: regular and
/// @notice temporary.
/// @notice
/// @notice [regular] account is a separate smart contract
/// @notice (Account.sol) that is meant to be used long-term. User can
/// @notice deposit $KLAY to account and use it to pay for Orakl Network
/// @notice services. More details about [regular] account are
/// @notice described at `Account` smart contract.
/// @notice
/// @notice [temporary] account is created for one-time-use of Orakl
/// @notice Network services. Consumer can send $KLAY together with
/// @notice request to Orakl Network to pay for the service. All
/// @notice operations related to [temporary] account are implemented
/// @notice in the Prepayment contract.
contract Prepayment is Ownable, IPrepayment, ITypeAndVersion {
    uint64 constant TEMPORARY_ACCOUNT_NONCE = 1;
    uint8 public constant MIN_RATIO = 0;
    uint8 public constant MAX_RATIO = 100;
    uint8 private sBurnFeeRatio = 50; // %
    uint8 private sProtocolFeeRatio = 5; // %

    address private sProtocolFeeRecipient;

    // Coordinator
    ICoordinatorBase[] public sCoordinators;

    /* coordinator address */
    /* association */
    mapping(address => bool) private sIsCoordinator;

    // Account
    uint64 private sCurrentAccId;

    /* accId */
    /* Account */
    mapping(uint64 => Account) private sAccIdToAccount;

    mapping(uint64 => bool) private sIsTemporaryAccount;

    struct TemporaryAccount {
        uint256 balance;
        address owner;
        IAccount.AccountType accType;
    }

    /* accId */
    /* TemporaryAccount */
    mapping(uint64 => TemporaryAccount) private sAccIdToTmpAcc;

    error PendingRequestExists();
    error InvalidCoordinator();
    error InvalidAccount();
    error MustBeAccountOwner();
    error RatioOutOfBounds();
    error FailedToDeposit();
    error FailedToWithdraw(uint64 accId);
    error CoordinatorExists();
    error InsufficientBalance();
    error BurnFeeFailed();
    error OperatorFeeFailed();
    error ProtocolFeeFailed();
    error TooHighFeeRatio();
    error FailedToWithdrawFromTemporaryAccount(uint64 accId);

    event AccountCreated(
        uint64 indexed accId,
        address account,
        address owner,
        IAccount.AccountType accType
    );
    event TemporaryAccountCreated(uint64 indexed accId, address owner);
    event AccountCanceled(uint64 indexed accId, address to, uint256 amount);
    event AccountBalanceIncreased(uint64 indexed accId, uint256 oldBalance, uint256 newBalance);
    event AccountBalanceDecreased(uint64 indexed accId, uint256 oldBalance, uint256 newBalance);
    event AccountConsumerAdded(uint64 indexed accId, address consumer);
    event AccountConsumerRemoved(uint64 indexed accId, address consumer);
    event BurnRatioSet(uint8 ratio);
    event ProtocolFeeRatioSet(uint8 ratio);
    event CoordinatorAdded(address coordinator);
    event CoordinatorRemoved(address coordinator);
    event AccountOwnerTransferRequested(uint64 indexed accId, address from, address to);
    event AccountOwnerTransferred(uint64 indexed accId, address from, address to);
    event BurnedFee(uint64 indexed accId, uint256 amount);
    event AccountDetailUpdated(
        uint64 indexed accId,
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount
    );
    event AccountFeeRatioSet(uint64 indexed accId, uint256 discount);
    event AccountPeriodReqIncreased(uint64 indexed accId);
    event AccountSubscriptionPaidSet(uint256 accId);

    /**
     * @dev The modifier is only for [regular] account. If called with
     * @dev account ID assigned to [temporary] account, then the
     * @dev transaction will be reverted because there is no
     * @dev associated `Account` contract.
     */
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
     * @param ratio in a range 0 - 100 % of a fee to be burnt
     */
    function setBurnFeeRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_RATIO || ratio > MAX_RATIO) {
            revert RatioOutOfBounds();
        }

        if ((ratio + sProtocolFeeRatio) > 100) {
            revert TooHighFeeRatio();
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
     * @param ratio in a range 0 - 100 % of a fee to be burnt
     */
    function setProtocolFeeRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_RATIO || ratio > MAX_RATIO) {
            revert RatioOutOfBounds();
        }
        if ((ratio + sBurnFeeRatio) > 100) {
            revert TooHighFeeRatio();
        }
        sProtocolFeeRatio = ratio;
        emit ProtocolFeeRatioSet(ratio);
    }

    /**
     * @notice Get address of protocol fee recipient.
     */
    function getProtocolFeeRecipient() external view returns (address) {
        return sProtocolFeeRecipient;
    }

    /**
     * @notice Update address of protocol fee recipient that will
     * @notice receive protocol fees.
     * @param protocolFeeRecipient - address of protocol fee recipient
     */
    function setProtocolFeeRecipient(address protocolFeeRecipient) external onlyOwner {
        sProtocolFeeRecipient = protocolFeeRecipient;
    }

    /**
     * @notice Get addresses of all registered coordinators in Prepayment.
     */
    function getCoordinators() external view returns (ICoordinatorBase[] memory) {
        return sCoordinators;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function isValidAccount(uint64 accId, address consumer) external view returns (bool) {
        Account account = sAccIdToAccount[accId];
        bool isValidRegular = address(account) != address(0) && account.getNonce(consumer) != 0;
        bool isValidTemporary = sAccIdToTmpAcc[accId].owner == consumer;

        return isValidRegular || isValidTemporary;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getBalance(uint64 accId) external view returns (uint256 balance) {
        return sAccIdToAccount[accId].getBalance();
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
        returns (
            uint256 balance,
            uint64 reqCount,
            address owner,
            address[] memory consumers,
            IAccount.AccountType accType
        )
    {
        Account account = sAccIdToAccount[accId];

        if (address(account) != address(0)) {
            // [regular] account
            return account.getAccount();
        } else if (sIsTemporaryAccount[accId]) {
            // [temporary] account
            TemporaryAccount memory tmpAccConfig = sAccIdToTmpAcc[accId];
            return (
                tmpAccConfig.balance,
                0,
                tmpAccConfig.owner,
                consumers,
                IAccount.AccountType.TEMPORARY
            );
        } else {
            revert InvalidAccount();
        }
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccountOwner(uint64 accId) external view returns (address) {
        return sAccIdToAccount[accId].getOwner();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getNonce(uint64 accId, address consumer) external view returns (uint64) {
        return sAccIdToAccount[accId].getNonce(consumer);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccountDetail(
        uint64 accId
    ) external view returns (uint256, uint256, uint256, uint256) {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        return account.getAccountDetail();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getSubscriptionPaid(uint64 accId) external view returns (bool) {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        return account.getSubscriptionPaid();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function isValidReq(uint64 accId) external view returns (bool) {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        return account.isValidReq();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getFeeRatio(uint64 accId) external view returns (uint256) {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        return account.getFeeRatio();
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createAccount() external returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;

        Account acc = new Account(currentAccId, msg.sender, IAccount.AccountType.KLAY_REGULAR);
        sAccIdToAccount[currentAccId] = acc;

        emit AccountCreated(
            currentAccId,
            address(acc),
            msg.sender,
            IAccount.AccountType.KLAY_REGULAR
        );
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createFiatSubscriptionAccount(
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount,
        address accOwner
    ) external onlyOwner returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;

        Account acc = new Account(currentAccId, accOwner, IAccount.AccountType.FIAT_SUBSCRIPTION);
        sAccIdToAccount[currentAccId] = acc;

        acc.updateAccountDetail(startDate, period, reqPeriodCount, 0);

        emit AccountCreated(
            currentAccId,
            address(acc),
            accOwner,
            IAccount.AccountType.FIAT_SUBSCRIPTION
        );
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createKlaySubscriptionAccount(
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount,
        uint256 subscriptionPrice,
        address accOwner
    ) external onlyOwner returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;

        Account acc = new Account(currentAccId, accOwner, IAccount.AccountType.KLAY_SUBSCRIPTION);
        sAccIdToAccount[currentAccId] = acc;
        acc.updateAccountDetail(startDate, period, reqPeriodCount, subscriptionPrice);
        emit AccountCreated(
            currentAccId,
            address(acc),
            accOwner,
            IAccount.AccountType.KLAY_SUBSCRIPTION
        );
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */

    function createKlayDiscountAccount(
        uint256 feeRatio,
        address accOwner
    ) external onlyOwner returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;
        Account acc = new Account(currentAccId, accOwner, IAccount.AccountType.KLAY_DISCOUNT);
        sAccIdToAccount[currentAccId] = acc;
        acc.setFeeRatio(feeRatio);
        emit AccountCreated(
            currentAccId,
            address(acc),
            accOwner,
            IAccount.AccountType.KLAY_DISCOUNT
        );
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function updateAccountDetail(
        uint64 accId,
        uint256 startTime,
        uint256 period,
        uint256 periodReqCount,
        uint256 subscriptionPrice
    ) external onlyOwner {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        account.updateAccountDetail(startTime, period, periodReqCount, subscriptionPrice);
        emit AccountDetailUpdated(accId, startTime, period, periodReqCount);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function setSubscriptionPaid(uint64 accId) external onlyCoordinator {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        account.setSubscriptionPaid();
        emit AccountSubscriptionPaidSet(accId);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function setFeeRatio(uint64 accId, uint256 discount) external onlyOwner {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        account.setFeeRatio(discount);
        emit AccountFeeRatioSet(accId, discount);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function increaseSubReqCount(uint64 accId) external onlyCoordinator {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) revert InvalidAccount();
        account.increaseSubReqCount();
        emit AccountPeriodReqIncreased(accId);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createTemporaryAccount(address owner) external returns (uint64) {
        uint64 currentAccId = sCurrentAccId + 1;
        sCurrentAccId = currentAccId;
        sAccIdToTmpAcc[currentAccId] = TemporaryAccount({
            balance: 0,
            owner: owner,
            accType: IAccount.AccountType.TEMPORARY
        });
        sIsTemporaryAccount[currentAccId] = true;

        emit TemporaryAccountCreated(currentAccId, owner);
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function requestAccountOwnerTransfer(
        uint64 accId,
        address requestedOwner
    ) external onlyAccountOwner(accId) {
        sAccIdToAccount[accId].requestAccountOwnerTransfer(requestedOwner);
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
        delete sAccIdToAccount[accId];

        account.cancelAccount(to);

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
        uint256 balance = sAccIdToTmpAcc[accId].balance;
        uint256 updatedBalance = balance + msg.value;
        sAccIdToTmpAcc[accId].balance = updatedBalance;
        emit AccountBalanceIncreased(accId, balance, updatedBalance);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function withdraw(uint64 accId, uint256 amount) external onlyAccountOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        (bool sent, uint256 balance) = sAccIdToAccount[accId].withdraw(amount);
        if (!sent) {
            revert FailedToWithdraw(accId);
        }

        emit AccountBalanceDecreased(accId, balance + amount, balance);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function withdrawTemporary(uint64 accId, address payable to) external {
        if (pendingRequestExistsTemporary(accId)) {
            revert PendingRequestExists();
        }

        TemporaryAccount memory tmpAcc = sAccIdToTmpAcc[accId];
        if (tmpAcc.owner != msg.sender) {
            revert MustBeAccountOwner();
        }

        uint256 balance = tmpAcc.balance;
        if (balance == 0) {
            revert InsufficientBalance();
        }

        delete sAccIdToTmpAcc[accId];

        (bool sent, ) = to.call{value: balance}("");
        if (!sent) {
            revert FailedToWithdrawFromTemporaryAccount(accId);
        }

        emit AccountBalanceDecreased(accId, balance, 0);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function chargeFee(uint64 accId, uint256 amount) external onlyCoordinator returns (uint256) {
        Account account = sAccIdToAccount[accId];
        uint256 balance = account.getBalance();

        if (balance < amount) {
            revert InsufficientBalance();
        }

        uint256 burnFee = (amount * sBurnFeeRatio) / 100;
        uint256 protocolFee = (amount * sProtocolFeeRatio) / 100;
        account.chargeFee(burnFee, protocolFee, sProtocolFeeRecipient);

        emit AccountBalanceDecreased(accId, balance, balance - burnFee - protocolFee);
        emit BurnedFee(accId, burnFee);

        return amount - burnFee - protocolFee;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function chargeOperatorFee(
        uint64 accId,
        uint256 operatorFee,
        address operatorFeeRecipient
    ) external onlyCoordinator {
        Account account = sAccIdToAccount[accId];
        uint256 balance = account.getBalance();

        if (balance < operatorFee) {
            revert InsufficientBalance();
        }

        account.chargeOperatorFee(operatorFee, operatorFeeRecipient);
        emit AccountBalanceDecreased(accId, balance, balance - operatorFee);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function chargeFeeTemporary(
        uint64 accId
    ) external onlyCoordinator returns (uint256 totalAmount, uint256 operatorAmount) {
        uint256 amount = sAccIdToTmpAcc[accId].balance;
        delete sAccIdToTmpAcc[accId];

        uint256 burnFee = (amount * sBurnFeeRatio) / 100;
        uint256 protocolFee = (amount * sProtocolFeeRatio) / 100;
        uint256 operatorFee = amount - burnFee - protocolFee;

        if (burnFee > 0) {
            (bool sent, ) = address(0x000000000000000000000000000000000000dEaD).call{
                value: burnFee
            }("");
            if (!sent) {
                revert BurnFeeFailed();
            }
        }

        if (protocolFee > 0) {
            (bool sent, ) = sProtocolFeeRecipient.call{value: protocolFee}("");
            if (!sent) {
                revert ProtocolFeeFailed();
            }
        }

        emit AccountBalanceDecreased(accId, amount, 0);
        emit BurnedFee(accId, burnFee);

        return (amount, operatorFee);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function chargeOperatorFeeTemporary(
        uint256 operatorFee,
        address operatorFeeRecipient
    ) external onlyCoordinator {
        (bool sent, ) = operatorFeeRecipient.call{value: operatorFee}("");
        if (!sent) {
            revert OperatorFeeFailed();
        }
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
            // [regular] account keeps track of nonce per each
            // consumer. Every consumer request should increase nonce
            // by one.
            return account.increaseNonce(consumer);
        } else {
            // [temporary] account can create only a single request
            // per its lifetime, therefore we do not keep track of
            // nonce and always return 1.
            return TEMPORARY_ACCOUNT_NONCE;
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
        for (uint256 i = 0; i < coordinatorsLength; ++i) {
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

    /**
     * @inheritdoc IPrepayment
     */
    function pendingRequestExistsTemporary(uint64 accId) public view returns (bool) {
        address consumer = sAccIdToTmpAcc[accId].owner;
        uint256 coordinatorsLength = sCoordinators.length;
        for (uint256 j = 0; j < coordinatorsLength; j++) {
            if (sCoordinators[j].pendingRequestExists(consumer, accId, TEMPORARY_ACCOUNT_NONCE)) {
                return true;
            }
        }

        return false;
    }
}
