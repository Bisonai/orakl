// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./IAccount.sol";

interface IPrepayment {
    /// READ-ONLY FUNCTIONS /////////////////////////////////////////////////////

    /**
     * @notice Returns `true` when a `consumer` is registered under
     * @notice `accId`, otherwise returns `false`.
     * @dev This function can be used for checking validity of both
     * @dev [regular] and [temporary] account.
     * @param accId - ID of the account
     */
    function isValidAccount(uint64 accId, address consumer) external view returns (bool);

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
     * @return balance - $KLAY balance of the account in juels.
     * @return reqCount - number of requests for this account, determines fee tier.
     * @return owner - owner of the account.
     * @return consumers - list of consumer address which are able to use this account.
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
        );

    /**
     * @notice Get address of account owner.
     * @dev This function is meant to be used only for [regular]
     * @dev account.
     * @param accId - ID of the account
     */
    function getAccountOwner(uint64 accId) external returns (address);

    /**
     * @notice Get nonce for specified `consumer` in account denoted by `accId`.
     * @dev This function is meant to be used only for [regular]
     * @dev account. [temporary] account does not have a notion of a nonce.
     * @dev When an invalid `accId` is passed, transaction is
     * @dev reverted. When an invalid `consumer` is passed, 0 zero
     * @dev nonce is returned that represents an unregistered consumer.
     * @param accId - ID of the account
     * @param consumer - consumer address
     */
    function getNonce(uint64 accId, address consumer) external view returns (uint64);

    /*
     * @notice Check to see if there exists a request commitment
     * @notice for all consumers and coordinators for a given
     * @notice [permanent] account.
     * @dev Use to reject account cancelation while outstanding
     * @dev request are present.
     * @param accId - ID of the account
     * @return true if there exists at least one unfulfilled request for the account, false
     * otherwise.
     */
    function pendingRequestExists(uint64 accId) external view returns (bool);

    /*
     * @notice Check to see if there exists a request commitment
     * @notice for an account owner of [temporary] account across
     * @notice all coordinators.
     * @dev Use to reject balance withdrawal while outstanding
     * @dev request are present.
     * @param accId - ID of the account
     * @return true if there exists at least one unfulfilled request for the account, false
     * otherwise.
     */
    function pendingRequestExistsTemporary(uint64 accId) external view returns (bool);

    /// STATE-ALTERING FUNCTIONS ////////////////////////////////////////////////

    /**
     * @notice Create a [regular] account.
     * @dev This function deploys a new `Account` contract (defined at
     * @dev Account.sol) and connect it with the `Prepayment` contract.
     * @dev You can add or remove the consumer dynamically with
     * @dev `addConsumer` or `removeConsumer` functions,
     * @dev respectively. To fund the account, use deposit function.
     * @return accId - A unique account id
     */
    function createAccount() external returns (uint64);

    function createFiatSubscriptionAccount(
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount,
        address accOwner
    ) external returns (uint64);

    function createKlaySubscriptionAccount(
        uint256 startDate,
        uint256 period,
        uint256 reqPeriodCount,
        uint256 subscriptionPrice,
        address accOwner
    ) external returns (uint64);

    function createKlayDiscountAccount(
        uint256 feeRatio,
        address accOwner
    ) external returns (uint64);

    /**
     * @notice Create a temporary account to be used with a single
     * @notice service request.
     * @param - account owner
     * @return accId - A unique account id
     */
    function createTemporaryAccount(address owner) external returns (uint64);

    /**
     * @notice Request account owner transfer.
     * @dev Only [regular] account owner can be transferred.
     * @param accId - ID of the account
     * @param newOwner - proposed new owner of the account
     */
    function requestAccountOwnerTransfer(uint64 accId, address newOwner) external;

    /**
     * @notice Accept account owner transfer.
     * @dev The function will revert inside of the
     * @dev `Account.acceptAccountOwnerTransfer` if original owner of
     * @dev `accId` has not requested the `msg.sender` to become the
     * @dev new owner.
     * @param accId - ID of the account
     */
    function acceptAccountOwnerTransfer(uint64 accId) external;

    /**
     * @notice Cancel account
     * @dev This function is meant to be used only for [regular]
     * @dev account. If there is any pending request, the account
     * @dev cannot be canceled.
     * @param accId - ID of the account
     * @param to - Where to send the remaining $KLAY to
     */
    function cancelAccount(uint64 accId, address to) external;

    /**
     * @notice Add a consumer to an account.
     * @dev This function is meant to be used only for [regular]
     * @dev account. If called with [temporary] account, the
     * @dev transaction will be reverted.
     * @param accId - ID of the account
     * @param consumer - New consumer which can use the account
     */
    function addConsumer(uint64 accId, address consumer) external;

    /**
     * @notice Remove a consumer from a account.
     * @dev This function is meant to be used only for [regular]
     * @dev account. If called with [temporary] account, the
     * @dev transaction will be reverted.
     * @param accId - ID of the account
     * @param consumer - Consumer to remove from the account
     */
    function removeConsumer(uint64 accId, address consumer) external;

    /**
     * @notice Deposit $KLAY to [regular] account.
     * @notice Anybody can deposit $KLAY, there are no restrictions.
     * @param accId - ID of the account
     */
    function deposit(uint64 accId) external payable;

    /**
     * @notice Deposit $KLAY to [temporary] account.
     * @notice Anybody can deposit $KLAY, there are no restrictions.
     * @param accId - ID of the account
     */
    function depositTemporary(uint64 accId) external payable;

    /**
     * @notice Withdraw $KLAY from [regular] account.
     * @dev Account owner can withdraw $KLAY only when there are no
     * @dev pending requests on any of associated consumers. If one tries
     * @dev to use it to withdraw $KLAY from [temporary] account,
     * @dev transaction will revert. Transaction reverts also on failure to
     * @dev withdraw tokens from account.
     * @param accId - ID of the account
     * @param amount - $KLAY amount to be withdrawn
     */
    function withdraw(uint64 accId, uint256 amount) external;

    /**
     * @notice Withdraw $KLAY from [temporary] account.
     * @dev Account owner can withdraw $KLAY only when there are no
     * @dev pending requests. Temporary account will be deleted upon
     * @dev successful withdrawal. Transaction reverts also on failure to
     * @dev withdraw tokens from account.
     * @param accId - ID of the account
     * @param to - recipient address
     */
    function withdrawTemporary(uint64 accId, address payable to) external;

    /**
     * @notice Burn part of fee and charge protocol fee for a service
     * connected to [regular] account.
     * @param accId - ID of the account
     * @param amount - $KLAY amount to be charged
     */
    function chargeFee(uint64 accId, uint256 amount) external returns (uint256);

    /**
     * @notice Charge operator fee for a service connected to
     * [temporary] account.
     * @param accId - ID of the account
     * @param operatorFee - amount of fee to be paid to operator fee
     * recipient
     * @param operatorFeeRecipient - address of operator fee recipient
     */
    function chargeOperatorFee(
        uint64 accId,
        uint256 operatorFee,
        address operatorFeeRecipient
    ) external;

    /**
     * @notice Burn part of fee and charge protocol fee for a service
     * connected to [temporary] account.
     * @dev Temporary account is deleted because we do not expect to use it again.
     * @param accId - ID of the account
     */
    function chargeFeeTemporary(
        uint64 accId
    ) external returns (uint256 totalAmount, uint256 operatorAmount);

    /**
     * @notice Charge operator fee for a service connected to
     * [temporary] account.
     * @param operatorFee - amount of fee to be paid to operator fee
     * recipient
     * @param operatorFeeRecipient - address of operator fee recipient
     */
    function chargeOperatorFeeTemporary(uint256 operatorFee, address operatorFeeRecipient) external;

    /**
     * @notice Increase nonce for consumer registered under accId.
     * @param accId - ID of the account
     * @param consumer - Address of consumer registered under accId
     */
    function increaseNonce(uint64 accId, address consumer) external returns (uint64);

    /*
     * @notice Add coordinator that will be able to charge account for
     * @notice the requested service.
     * @param coordinator - address of coordinator
     */
    function addCoordinator(address coordinator) external;

    /*
     * @notice Disable the coordinator from being able to charge
     * @notice accounts for its service.
     * @param coordinator - address of coordinator
     */
    function removeCoordinator(address coordinator) external;

    function getBurnFeeRatio() external view returns (uint8);

    function getProtocolFeeRatio() external view returns (uint8);

    function getAccountDetail(
        uint64 accId
    ) external view returns (uint256, uint256, uint256, uint256);

    function getSubscriptionPaid(uint64 accId) external view returns (bool);

    function isValidReq(uint64 accId) external view returns (bool);

    function getFeeRatio(uint64 accId) external view returns (uint256);

    function updateAccountDetail(
        uint64 accId,
        uint256 startTime,
        uint256 endTime,
        uint256 periodReqCount,
        uint256 subscriptionPrice
    ) external;

    function setSubscriptionPaid(uint64 accId) external;

    function setFeeRatio(uint64 accId, uint256 disCount) external;

    function increaseSubReqCount(uint64 accId) external;
}
