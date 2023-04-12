// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IAccount.sol";
import "./interfaces/ITypeAndVersion.sol";

/// @title Orakl Network Account
/// @author Bisonai
/// @notice Every consumer has to create an account in order to be able to setup
/// @notice TODO
/// TODO selfdestruct
/// @dev
contract Account is IAccount, ITypeAndVersion {
    uint64 private sAccId;

    // Account information
    address private sOwner; // Owner can fund/withdraw/cancel the acc
    address private sRequestedOwner; // For safely transferring acc ownership
    uint256 private sBalance; // Common KLAY balance used for all consumer requests
    uint64 private sReqCount; // For fee tiers
    address[] private sConsumers;

    address private sPaymentSolution;

    /* consumer */
    /* nonce */
    mapping(address => uint64) private sConsumerToNonce;

    error MustBeRequestedOwner(address requestedOwner);
    error MustBeAccountOwner(address owner);
    error MustBePaymentSolution(address paymentSolution);
    error InsufficientBalance();

    event AccountTransferRequested(uint64 indexed accId, address from, address to);
    event AccountTransferred(uint64 indexed accId, address from, address to);
    event AccountCanceled(uint64 indexed accId, address to, uint256 amount);

    modifier onlyAccountOwner() {
        if (msg.sender != sOwner) {
            revert MustBeAccountOwner(sOwner);
        }
        _;
    }

    modifier onlyPaymentSolution() {
        if (msg.sender != sPaymentSolution) {
            revert MustBePaymentSolution(sPaymentSolution);
        }
        _;
    }

    constructor(uint64 accId, address owner) {
        sAccId = accId;
        sOwner = owner;
        sPaymentSolution = msg.sender;
    }

    /**
     * @inheritdoc IAccount
     */
    function getAccountId() external view returns (uint64) {
        return sAccId;
    }

    /**
     * @inheritdoc IAccount
     */
    function getBalance() external view returns (uint256) {
        return sBalance;
    }

    /**
     * @inheritdoc IAccount
     */
    function getOwner() external view returns (address) {
        return sOwner;
    }

    /**
     * @inheritdoc IAccount
     */
    function getConsumers() external view returns (address[] memory) {
        return sConsumers;
    }

    /**
     * @inheritdoc IAccount
     */
    function getRequestedOwner() external view returns (address) {
        return sRequestedOwner;
    }

    /**
     * @inheritdoc IAccount
     */
    function getNonce(address consumer) external view returns (uint64) {
        return sConsumerToNonce[consumer];
    }

    /**
     * @inheritdoc IAccount
     */
    function getPaymentSolution() external view returns (address) {
        return sPaymentSolution;
    }

    /**
     * @inheritdoc IAccount
     */
    function requestAccountTransfer(address requestedOwner) external onlyAccountOwner {
        // Proposing the address(0) would never be claimable so no
        // need to check.
        if (sRequestedOwner != requestedOwner) {
            sRequestedOwner = requestedOwner;
            emit AccountTransferRequested(sAccId, msg.sender, requestedOwner);
        }
    }

    /**
     * @inheritdoc IAccount
     */
    function acceptAccountTransfer() external {
        if (sRequestedOwner != msg.sender) {
            revert MustBeRequestedOwner(sRequestedOwner);
        }

        address oldOwner = sOwner;
        sOwner = msg.sender;
        sRequestedOwner = address(0);

        emit AccountTransferred(sAccId, oldOwner, msg.sender);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Account v0.1";
    }

    function cancelAccount(address to) external onlyPaymentSolution {
        emit AccountCanceled(sAccId, to, sBalance);
        selfdestruct(payable(to));
    }
}
