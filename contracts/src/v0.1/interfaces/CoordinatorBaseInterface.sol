// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface CoordinatorBaseInterface {
    /**
   * @notice Check to see if there exists a request commitment consumers
   * for all consumers and keyhashes for a given sub.
   * @param subId - ID of the subscription
   * @return true if there exists at least one unfulfilled request for the subscription, false
   * otherwise.
   */
  function pendingRequestExists (uint64 subId,
        address consumer,
        uint64 nonce
    ) external view returns (bool);
}
