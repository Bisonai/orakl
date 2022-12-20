// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface SubscriptionInterface {
    ///// from VRFcoordinator /////

    function getSubscription(uint64 subId) external view
        returns (uint96 balance, uint64 reqCount, address owner, address[] memory consumers);

    function createSubscription() external returns (uint64);

    function requestSubscriptionOwnerTransfer(uint64 subId, address newOwner) external;

    function acceptSubscriptionOwnerTransfer(uint64 subId) external;

    function removeConsumer(uint64 subId, address consumer) external;

    function addConsumer(uint64 subId, address consumer) external;

    function cancelSubscription(uint64 subId, address to) external;

    ///// added interfaces /////
    function increaseSubBalance(uint64 subId,uint96 amount) external;

    function decreaseSubBalance(uint64 subId,uint96 amount) external;
}
