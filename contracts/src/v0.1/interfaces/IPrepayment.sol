// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./ICoordinatorBase.sol";

interface IPrepayment {
    /// READ-ONLY FUNCTIONS /////////////////////////////////////////////////////

    /**
     * @notice Get an account information.
     * @param accId - ID of the account
     * @return balance - KLAY balance of the account in juels.
     * @return reqCount - number of requests for this account, determines fee tier.
     * @return owner - owner of the account.
     * @return consumers - list of consumer address which are able to use this account.
     */
    function getAccount(
        uint64 accId
    )
        external
        view
        returns (uint256 balance, uint64 reqCount, address owner, address[] memory consumers);

    /**
     * @notice Get address of account owner.
     * @param accId - ID of the account
     */
    function getAccountOwner(uint64 accId) external returns (address);

    /**
     * @notice Get address of protocol fee recipient.
     */
    function getProtocolFeeRecipient() external view returns (address);

    /**
     * @notice Get addresses of all registered coordinators in Prepayment.
     */
    function getCoordinators() external view returns (ICoordinatorBase[] memory);

    /**
     * @notice Get nonce for specified `consumer` in account denoted by `accId`.
     * @param accId - ID of the account
     * @param consumer - consumer address
     */
    function getNonce(uint64 accId, address consumer) external view returns (uint64);

    /*
     * @notice Check to see if there exists a request commitment consumers
     * for all consumers and keyhashes for a given acc.
     * @param accId - ID of the account
     * @return true if there exists at least one unfulfilled request for the account, false
     * otherwise.
     */
    function pendingRequestExists(uint64 accId) external view returns (bool);

    /// STATE-ALTERING FUNCTIONS ////////////////////////////////////////////////

    /**
     * @notice Update address of protocol fee recipient that will
     * @notice receive protocol fees.
     * @param protocolFeeRecipient - address of protocol fee recipient
     */
    function setProtocolFeeRecipient(address protocolFeeRecipient) external;

    /**
     * @notice Create an account.
     * @return accId - A unique account id.
     * @dev You can manage the consumer set dynamically with addConsumer/removeConsumer.
     * @dev Note to fund the account, use deposit function.
     */
    function createAccount() external returns (uint64);

    /**
     * @notice Request account owner transfer.
     * @param accId - ID of the account
     * @param newOwner - proposed new owner of the account
     */
    function requestAccountOwnerTransfer(uint64 accId, address newOwner) external;

    /**
     * @notice Request account owner transfer.
     * @param accId - ID of the account
     * @dev will revert if original owner of accId has
     * not requested that msg.sender become the new owner.
     */
    function acceptAccountOwnerTransfer(uint64 accId) external;

    /**
     * @notice Remove a consumer from a account.
     * @param accId - ID of the account
     * @param consumer - Consumer to remove from the account
     */
    function removeConsumer(uint64 accId, address consumer) external;

    /**
     * @notice Add a consumer to an account.
     * @param accId - ID of the account
     * @param consumer - New consumer which can use the account
     */
    function addConsumer(uint64 accId, address consumer) external;

    /**
     * @notice Cancel account
     * @param accId - ID of the account
     * @param to - Where to send the remaining KLAY to
     */
    function cancelAccount(uint64 accId, address to) external;

    /**
     * @notice Deposit KLAY to account.
     * @notice Anybody can deposit KLAY, there are no restrictions.
     * @param accId - ID of the account
     */
    function deposit(uint64 accId) external payable;

    /**
     * @notice Withdraw KLAY from account.
     * @notice Only account owner can withdraw KLAY.
     * @param accId - ID of the account
     * @param amount - KLAY amount to be withdrawn
     */
    function withdraw(uint64 accId, uint256 amount) external;

    /**
     * @notice Charge fee from service connected to account.
     * @param accId - ID of the account
     * @param amount - KLAY amount to be charged
     * @param operatorFeeRecipient - address of operator that receives fee
     */
    function chargeFee(uint64 accId, uint256 amount, address operatorFeeRecipient) external;

    /**
     * @notice Increase nonce for consumer registered under accId.
     * @param accId - ID of the account
     * @param consumer - Address of consumer registered under accId
     */
    function increaseNonce(uint64 accId, address consumer) external returns (uint64);

    /*
     * @notice Add coordinator to be able to charge using Prepayment method.
     * @param coordinator - address of coordinator
     */
    function addCoordinator(address coordinator) external;

    /*
     * @notice Block coordinator from using Prepayment method.
     * @param coordinator - address of coordinator
     */
    function removeCoordinator(address coordinator) external;
}
