// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IL2Endpoint {
    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external returns (uint256);
}
