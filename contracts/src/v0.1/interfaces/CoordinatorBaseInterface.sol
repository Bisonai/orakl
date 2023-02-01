// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface CoordinatorBaseInterface {
    /**
     * @notice Check to see if there exists a request commitment consumers
     * for all consumers and keyhashes for a given acc.
     * @param accId - ID of the account
     * @return true if there exists at least one unfulfilled request for the account, false
     * otherwise.
     */
    function pendingRequestExists(
        address consumer,
        uint64 accId,
        uint64 nonce
    ) external view returns (bool);
}
