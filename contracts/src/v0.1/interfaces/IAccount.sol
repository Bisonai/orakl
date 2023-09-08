// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IAccount {
    /// READ-ONLY FUNCTIONS /////////////////////////////////////////////////////
    enum AccountType {
        TEMPORARY,
        FIAT_SUBSCRIPTION,
        KLAY_SUBSCRIPTION,
        KLAY_DISCOUNT,
        KLAY_REGULAR
    }

    /**
     * @notice Get an account information.
     * @return balance - KLAY balance of the account in juels.
     * @return reqCount - number of requests for this account, determines fee tier.
     * @return owner - owner of the account.
     * @return consumers - list of consumer address which are able to use this account.
     * @return accType
     */
    function getAccount()
        external
        view
        returns (
            uint256 balance,
            uint64 reqCount,
            address owner,
            address[] memory consumers,
            AccountType accType
        );

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
     * @notice Return the number of requests created through the
     * @notice account.
     * @return number of requests
     */
    function getReqCount() external returns (uint64);

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
    function getNonce(address consumer) external view returns (uint64);

    /**
     * @notice Return the address of payment solution associated with account.
     * @return address of payment solution
     */
    function getPaymentSolution() external view returns (address);

    /// STATE-ALTERING FUNCTIONS ////////////////////////////////////////////////

    /**
     * @notice Increase nonce for given consumer.
     * @param consumer - Address of consumer
     */
    function increaseNonce(address consumer) external returns (uint64);

    /**
     * @notice Request account owner transfer.
     * @param newOwner - proposed new owner of the account
     */
    function requestAccountOwnerTransfer(address newOwner) external;

    /**
     * @notice Request account owner transfer.
     * @dev will revert if original owner of accId has
     * not requested that msg.sender become the new owner.
     * @param newOwner - proposed new owner of the account
     */
    function acceptAccountOwnerTransfer(address newOwner) external;

    /**
     * @notice Add a consumer to an account.
     * @param consumer - New consumer which can use the account
     */
    function addConsumer(address consumer) external;

    /**
     * @notice Remove a consumer from a account.
     * @param consumer - Consumer to remove from the account
     */
    function removeConsumer(address consumer) external;

    /**
     * @notice Withdraw KLAY from account.
     * @dev Only account owner can withdraw KLAY.
     * @param amount - KLAY amount to be withdrawn
     */
    function withdraw(uint256 amount) external returns (bool, uint256);

    /**
     * @notice Burn part of fee and charge protocol fee for a service
     * connected to account.
     * @param burnFee - $KLAY amount to be burnt
     * @param protocolFee - $KLAY amount to be sent to protocol fee recipient
     * @param protocolFeeRecipient - address of Orakl Network
     */
    function chargeFee(uint256 burnFee, uint256 protocolFee, address protocolFeeRecipient) external;

    /**
     * @notice Charge operator fee for a service connected to account.
     * @param operatorFee - $KLAY amount to be send to oracle operator
     * fee recipient
     * @param operatorFeeRecipient - address of Orakl Network
     */
    function chargeOperatorFee(uint256 operatorFee, address operatorFeeRecipient) external;

    /**
     * @notice Destroy the smart contract and send the remaining $KLAY
     * @notice to `to` address.
     * @param to - Where to send the remaining KLAY to
     */
    function cancelAccount(address to) external;

    function getAccountDetail() external view returns (uint256, uint256, uint256, uint256);

    function getSubscriptionPaid() external view returns (bool);

    function updateAccountDetail(
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount,
        uint256 subscriptionPrice
    ) external;

    function setSubscriptionPaid() external;

    function isValidReq() external view returns (bool);

    function getFeeRatio() external view returns (uint256);

    function setFeeRatio(uint256 disCount) external;

    function increaseSubReqCount() external;
}
