// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IAccount {
    /**
     * @notice Return an account ID that is associated with this account.
     * @return account ID
     */
    function getAccountId() external returns (uint64);

    /**
     * @notice Return an amount of KLAY held in the Account contract.
     * @return balance of account
     */
    function getBalance() external returns (uint256);

    /**
     * @notice Return the current owner of account.
     * @return owner address
     */
    function getOwner() external returns (address);

    /**
     * @notice Return the consumers assigned to the account.
     * @return list of consumer addresses
     */
    function getConsumers() external returns (address[] memory);

    /**
     * @notice Return the requested owner of account.
     * @return requested owner address
     */
    function getRequestedOwner() external returns (address);

    /**
     * @notice Return the current nonce of given consumer.
     * @return consumer nonce
     */
    function getConsumerNonce(address consumer) external view returns (uint64);

    function getPaymentSolution() external view returns (address);

    /**
     * @notice Return nonce value.
     * @param consumer - Address of consumer registered under accId
     * @param accId - ID of the account
     */
    /* function nonce(address consumer, uint64 accId) external view returns (uint64); */

    /// THE FOLLOWING FUNCTIONS CHANGE THE STATE OF ACCOUNT.

    /**
     * @notice Increase nonce for consumer registered under accId.
     * @param consumer - Address of consumer registered under accId
     * @param accId - ID of the account
     */
    /* function increaseNonce(address consumer, uint64 accId) external returns (uint64); */

    /**
     * @notice Request account owner transfer.
     * @param newOwner - proposed new owner of the account
     */
    function requestAccountTransfer(address newOwner) external;

    /**
     * @notice Request account owner transfer.
     * @dev will revert if original owner of accId has
     * not requested that msg.sender become the new owner.
     */
    function acceptAccountTransfer() external;

    /**
     * @notice Add a consumer to an account.
     * @param accId - ID of the account
     * @param consumer - New consumer which can use the account
     */
    /* function addConsumer(uint64 accId, address consumer) external; */

    /**
     * @notice Remove a consumer from a account.
     * @param accId - ID of the account
     * @param consumer - Consumer to remove from the account
     */
    /* function removeConsumer(uint64 accId, address consumer) external; */

    /**
     * @notice Deposit KLAY to account.
     * @dev Anybody can deposit KLAY, there are no restrictions.
     */
    /* function deposit() external payable; */

    /**
     * @notice Withdraw KLAY from account.
     * @dev Only account owner can withdraw KLAY.
     * @param amount - KLAY amount to be withdrawn
     */
    /* function withdraw(uint256 amount) external; */
}
