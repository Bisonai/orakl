pragma solidity ^0.8.16;
interface IRegistry {
    struct Account {
        uint256 accId;
        uint256 chainId;
        address owner;
        address feePayer;
        address[100] consumers;
        uint8 consumerCount;
    }

    function getAccount(uint256 _accId) external view returns (Account memory);

    function getAccountsByChain(uint256 _chainId) external view returns (Account[] memory);

    function getAccountsByOwner(address _owner) external view returns (Account[] memory);
}