// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./Account.sol";
import "./interfaces/IAccount.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";

contract Prepayment is Ownable, IPrepayment, ITypeAndVersion {
    uint8 public constant MIN_BURN_RATIO = 0;
    uint8 public constant MAX_BURN_RATIO = 100;

    uint8 private sBurnRatio = 20; // %
    uint64 private sCurrentAccId;

    ICoordinatorBase[] public sCoordinators;

    /* accId */
    /* Account */
    mapping(uint64 => Account) private sAccIdToAccount;

    /* coordinator address */
    /* association */
    mapping(address => bool) private sIsCoordinator;

    error PendingRequestExists();
    error InvalidAccount();
    error MustBeAccountOwner(address owner);
    error InvalidBurnRatio();

    event AccountCreated(uint64 indexed accId, address account, address owner);
    event AccountCanceled(uint64 indexed accId, address to, uint256 amount);

    /*     event AccountBalanceIncreased(uint64 indexed accId, uint256 oldBalance, uint256 newBalance); */
    /*     event AccountBalanceDecreased( */
    /*         uint64 indexed accId, */
    /*         uint256 oldBalance, */
    /*         uint256 newBalance, */
    /*         uint256 burnAmount */
    /*     ); */
    event AccountConsumerAdded(uint64 indexed accId, address consumer);
    event AccountConsumerRemoved(uint64 indexed accId, address consumer);
    /*     event NodeOperatorFundsWithdrawn(address to, uint256 amount); */
    event BurnRatioSet(uint16 ratio);

    modifier onlyAccountOwner(uint64 accId) {
        address owner = sAccIdToAccount[accId].getOwner();
        if (owner != msg.sender) {
            revert MustBeAccountOwner(owner);
        }
        _;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function createAccount() external returns (uint64) {
        sCurrentAccId++;
        uint64 currentAccId = sCurrentAccId;
        address[] memory consumers = new address[](0);

        Account acc = new Account(currentAccId, msg.sender);
        sAccIdToAccount[currentAccId] = acc;

        emit AccountCreated(currentAccId, address(acc), msg.sender);
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function cancelAccount(uint64 accId, address to) external onlyAccountOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        // TODO more cleanup ?

        Account account = sAccIdToAccount[accId];
        uint256 balance = account.getBalance();

        account.cancelAccount(to);
        delete sAccIdToAccount[accId];

        emit AccountCanceled(accId, to, balance);
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccount(uint64 accId) external view returns (address account) {
        account = address(sAccIdToAccount[accId]);
        if (account == address(0)) {
            revert InvalidAccount();
        }
    }

    function addConsumer(uint64 accId, address consumer) external onlyAccountOwner(accId) {
        sAccIdToAccount[accId].addConsumer(consumer);
        emit AccountConsumerAdded(accId, consumer);
    }

    function removeConsumer(uint64 accId, address consumer) external onlyAccountOwner(accId) {
        sAccIdToAccount[accId].removeConsumer(consumer);
        emit AccountConsumerRemoved(accId, consumer);
    }

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
        if (ratio < MIN_BURN_RATIO || ratio > MAX_BURN_RATIO) {
            revert InvalidBurnRatio();
        }
        sBurnRatio = ratio;
        emit BurnRatioSet(ratio);
    }

    /**
     * @inheritdoc IPrepayment
     * @dev Looping is bounded to MAX_CONSUMERS*(number of keyhashes).
     * @dev Used to disable subscription canceling while outstanding request are present.
     */
    function pendingRequestExists(uint64 accId) public view returns (bool) {
        /* AccountConfig memory accConfig = sAccIdToAccConfig[accId]; */

        Account acc = sAccIdToAccount[accId];
        address[] memory consumers = acc.getConsumers();
        uint256 consumersLength = consumers.length;
        uint256 coordinatorsLength = sCoordinators.length;

        for (uint256 i = 0; i < consumersLength; i++) {
            for (uint256 j = 0; j < coordinatorsLength; j++) {
                address consumer = consumers[i];
                uint64 nonce = acc.getNonce(consumer);
                if (sCoordinators[j].pendingRequestExists(consumer, accId, nonce)) {
                    return true;
                }
            }
        }
        return false;
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Prepayment v0.1";
    }
}
