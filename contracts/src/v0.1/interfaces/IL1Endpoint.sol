// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IL1Endpoint {
    function increaseBalance(uint256 _accId, uint256 _amount) external;
    function decreaseBalance(uint256 _accId, uint256 _amount) external;
}
