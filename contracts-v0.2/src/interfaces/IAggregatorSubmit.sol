// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IAggregator {
    function submit(int256 _submission) external;
}
