// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./Account.sol";
import "./interfaces/IAccount.sol";
import "./interfaces/ICoordinatorBase.sol";
import "./interfaces/IPrepayment.sol";
import "./interfaces/ITypeAndVersion.sol";

contract Prepayment is Ownable, IPrepayment, ITypeAndVersion {
    uint64 private sCurrentAccId;

    /* consumer */
    /* accId */
    /* nonce */
    mapping(address => mapping(uint64 => uint64)) private sConsumers;

    ICoordinatorBase[] public sCoordinators;

    /* accId */
    /* Account */
    mapping(uint64 => Account) private sAccIdToAccount;

    /* accId */
    /* AccountConfig */
    /* mapping(uint64 => AccountConfig) private sAccIdToAccConfig; */

    mapping(address => bool) private sIsCoordinator;

    /* struct AccountConfig { */
    /*     address owner; // Owner can fund/withdraw/cancel the acc. */
    /*     address requestedOwner; // For safely transferring acc ownership. */
    /*     // Maintains the list of keys in sConsumers. */
    /*     // We do this for 2 reasons: */
    /*     // 1. To be able to clean up all keys from sConsumers when canceling an account. */
    /*     // 2. To be able to return the list of all consumers in getAccount. */
    /*     // Note that we need the sConsumers map to be able to directly check if a */
    /*     // consumer is valid without reading all the consumers from storage. */
    /*     address[] consumers; */
    /* } */

    error PendingRequestExists();
    error InvalidAccount();
    error MustBeAccountOwner(address owner);

    event AccountCreated(uint64 indexed accId, address account, address owner);

    modifier onlyAccOwner(uint64 accId) {
        address owner = sAccIdToAccount[accId].getOwner();
        if (owner == address(0)) {
            revert InvalidAccount();
        }
        if (msg.sender != owner) {
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

        /* string memory accType = "reg"; */
        /* if (sIsCoordinator[msg.sender]) { */
        /*     accType = "tmp"; */
        /* } */

        Account acc = new Account(currentAccId, msg.sender);
        sAccIdToAccount[currentAccId] = acc;

        emit AccountCreated(currentAccId, address(acc), msg.sender);
        return currentAccId;
    }

    /**
     * @inheritdoc IPrepayment
     */
    function cancelAccount(uint64 accId, address to) external onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }

        // TODO cleanup

        sAccIdToAccount[accId].cancelAccount(to);
        delete sAccIdToAccount[accId];
    }

    /**
     * @inheritdoc IPrepayment
     */
    function getAccount(uint64 accId) external view {
        if (sAccIdToAccount[accId].getOwner() == address(0)) {
            revert InvalidAccount();
        }
        /* return ( */
        /*     sAccIdToAcc[accId].balance, */
        /*     sAccIdToAcc[accId].reqCount, */
        /*     sAccIdToAcc[accId].accType, */
        /*     sAccIdToAccConfig[accId].owner, */
        /*     sAccIdToAccConfig[accId].consumers */
        /* ); */
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
                uint64 nonce = acc.getConsumerNonce(consumer);
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
