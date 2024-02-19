// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "../libraries/Orakl.sol";

interface IL1Endpoint {
    function requestRandomWords(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        uint64 accId,
        address sender,
        uint256 l2RequestId
    ) external returns (uint256);

    function requestData(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId,
        Orakl.Request memory req
    ) external returns (uint256);
}
