// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IFeed {
    function submit(int256 answer) external;

    function name() external view returns (string memory);
}
