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
    uint8 private sBurnRatio = 20; // %
    uint8 private sProtocolFeeRatio = 5; // %

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

    error PendingRequestExists();
    error InvalidCoordinator();
    error InvalidAccount();
    error MustBeAccountOwner();
    error RatioOutOfBounds();
    error FailedToDeposit();
    error FailedToWithdraw();
    error CoordinatorExists();

    event AccountCreated(uint64 indexed accId, address account, address owner);
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
    /*     event NodeOperatorFundsWithdrawn(address to, uint256 amount); */
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

    function getCoordinators() external view returns (ICoordinatorBase[] memory) {
        return sCoordinators;
    }

    function getAccountOwner(uint64 accId) external view returns (address) {
        Account account = sAccIdToAccount[accId];
        return account.getOwner();
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
    function getAccountInfo(
        uint64 accId
    )
        external
        view
        returns (uint256 balance, uint64 reqCount, address owner, address[] memory consumers)
    {
        Account account = sAccIdToAccount[accId];
        if (address(account) == address(0)) {
            revert InvalidAccount();
        }

        return account.getAccountInfo();
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
    function getBurnRatio() external view returns (uint8) {
        return sBurnRatio;
    }

    /**
     * @notice The function allows to update a "burn ratio" that represents a
     * @notice partial amount of payment for the Orakl Network service that
     * @notice will be burnt.
     *
     * @param ratio in a range 0 - 100 % of a fee to be burnt
     */
    function setBurnRatio(uint8 ratio) external onlyOwner {
        if (ratio < MIN_RATIO || ratio > MAX_RATIO) {
            revert RatioOutOfBounds();
        }
        sBurnRatio = ratio;
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
    function increaseNonce(
        uint64 accId,
        address consumer
    ) external onlyCoordinator returns (uint64) {
        Account account = sAccIdToAccount[accId];
        return account.increaseNonce(consumer);
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
