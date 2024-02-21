// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "../libraries/Orakl.sol";

interface IL2Endpoint {
    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external returns (uint256);

    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numSubmission
    ) external returns (uint256);
}
