// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./ICoordinatorBase.sol";

interface IPrepayment {
    /// READ-ONLY FUNCTIONS /////////////////////////////////////////////////////

    /**
     * @notice Returns `true` when given `accId` is valid, otherwise reverts.
     * @dev This function can be used for checking validity of both
     * @dev [regular] and [temporary] account.
     * @param accId - ID of the account
     */
    function isValidAccount(uint64 accId) external view returns (bool);

    /**
     * @notice Returns the balance of given account.
     * @dev This function is meant to be used only for [regular]
     * @dev account. If invalid `accId` (ID not assigned to any
     * @dev account) is passed, zero balance will be always returned.
     * @param accId - ID of the account
     * @return balance of account
     */
    function getBalance(uint64 accId) external view returns (uint256);

    /**
     * @notice Return the number of requests created through the
     * @notice account.
     * @dev This function is meant to be used only for [regular]
     * @dev account.
     * @param accId - ID of the account
     * @return number of requests
     */
    function getReqCount(uint64 accId) external view returns (uint64);

    /**
     * @notice Get an account information.
     * @dev This function can be used for both [regular] and
     * @dev [temporary] account.
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
     * @dev This function is meant to be used only for [regular]
     * @dev account.
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
     * @dev This function is meant to be used for both [regular] and
     * @dev [temporary] account. In case of [regular] account, we keep
     * @dev track of nonces for every consumer inside of the
     * @dev account. [temporary] account is expected to be used only
     * @dev once, therefore we do not keep track of nonce, and always return 1.
     * @dev We do not check on validity of the `accId`, therefore when
     * @dev a an invalid `accId` is passed, nonce equal to 1 is returned.
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
     * @notice Create a temporary account for a single direct payment.
     * @return accId - A unique account id.
     */
    function createTemporaryAccount() external returns (uint64);

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
     * @notice Deposit KLAY to temporary account.
     * @notice Anybody can deposit KLAY, there are no restrictions.
     * @param accId - ID of the account
     */
    function depositTemporary(uint64 accId) external payable;

    /**
     * @notice Withdraw KLAY from account.
     * @notice Only account owner can withdraw KLAY.
     * @param accId - ID of the account
     * @param amount - KLAY amount to be withdrawn
     */
    function withdraw(uint64 accId, uint256 amount) external;

    /**
     * @notice Charge fee from [regular]  account for a service.
     * @param accId - ID of the account
     * @param amount - KLAY amount to be charged
     * @param operatorFeeRecipient - address of operator that receives fee
     */
    function chargeFee(uint64 accId, uint256 amount, address operatorFeeRecipient) external;

    /**
     * @notice Charge fee from [temporary] account for a service.
     * @param accId - ID of the account
     * @param operatorFeeRecipient - address of operator that receives fee
     */
    function chargeFee(uint64 accId, address operatorFeeRecipient) external returns (uint256);

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
