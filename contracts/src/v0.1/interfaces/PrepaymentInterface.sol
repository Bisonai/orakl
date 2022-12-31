// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface PrepaymentInterface {
    ///// from VRFcoordinator /////

    function getAccount(uint64 accId) external view
        returns (uint96 balance, uint64 reqCount, address owner, address[] memory consumers);

    function createAccount() external returns (uint64);

    function requestAccountOwnerTransfer(uint64 accId, address newOwner) external;

    function acceptAccountOwnerTransfer(uint64 accId) external;

    function removeConsumer(uint64 accId, address consumer) external;

    function addConsumer(uint64 accId, address consumer) external;

    function cancelAccount(uint64 accId, address to) external;

    ///// added interfaces /////
    function deposit(uint64 accId) payable external;

    function withdraw(uint64 accId, uint96 amount) external;

    function decreaseAccBalance(uint64 accId,uint96 amount) external;

    function getNonce(address consumer,uint64 accId) external view returns(uint64);

    function increaseNonce(address consumer,uint64 accId) external;

    function getAccOwner(uint64 accId)external returns(address owner);
}
