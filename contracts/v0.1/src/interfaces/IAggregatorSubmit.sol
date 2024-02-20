// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IAggregator {
    function submit(uint256 _roundId, int256 _submission) external;
}
