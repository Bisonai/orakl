// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IAggregator {
    function submit(int256 _submission) external;
}
