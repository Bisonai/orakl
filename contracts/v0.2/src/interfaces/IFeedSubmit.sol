// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeed {
    function submit(int256 _answer) external;

    function submit(int256 _answer, bytes memory _proof) external;
}
