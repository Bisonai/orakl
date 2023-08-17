// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IRegistry {
    struct Account {
        uint256 accId;
        uint256 chainId;
        address owner;
        address[] consumers;
        uint8 consumerCount;
        uint256 balance;
    }

    function deposit(uint256 _accId) external payable;

    function decreaseBalance(uint256 _accId, uint256 _amount) external;

    function getBalance(uint256 _accId) external view returns (uint256 balance);

    function accountInfo(uint256 _accId) external view returns (uint256 balance, address owner);

    function getConsumer(uint256 _accId) external view returns (address[] memory consumers);

    function getAccount(uint256 _accId) external view returns (Account memory);

    function isValidConsumer(uint256 _accId, address _consumer) external view returns (bool);
}
