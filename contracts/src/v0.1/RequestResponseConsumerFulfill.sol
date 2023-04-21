// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./RequestResponseConsumerBase.sol";

abstract contract RequestResponseConsumerFulfillUint256 is RequestResponseConsumerBase {
    function fulfillDataRequestUint256(uint256 requestId, uint256 response) internal virtual;

    function rawFulfillDataRequestUint256(
        uint256 requestId,
        uint256 response
    ) external verifyRawFulfillment {
        fulfillDataRequestUint256(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillInt256 is RequestResponseConsumerBase {
    function fulfillDataRequestInt256(uint256 requestId, int256 response) internal virtual;

    function rawFulfillDataRequestInt256(
        uint256 requestId,
        int256 response
    ) external verifyRawFulfillment {
        fulfillDataRequestInt256(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBool is RequestResponseConsumerBase {
    function fulfillDataRequestBool(uint256 requestId, bool response) internal virtual;

    function rawFulfillDataRequestBool(
        uint256 requestId,
        bool response
    ) external verifyRawFulfillment {
        fulfillDataRequestBool(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillString is RequestResponseConsumerBase {
    function fulfillDataRequestString(uint256 requestId, string memory response) internal virtual;

    function rawFulfillDataRequestString(
        uint256 requestId,
        string memory response
    ) external verifyRawFulfillment {
        fulfillDataRequestString(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBytes32 is RequestResponseConsumerBase {
    function fulfillDataRequestBytes32(uint256 requestId, bytes32 response) internal virtual;

    function rawFulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response
    ) external verifyRawFulfillment {
        fulfillDataRequestBytes32(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBytes is RequestResponseConsumerBase {
    function fulfillDataRequestBytes(uint256 requestId, bytes memory response) internal virtual;

    function rawFulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response
    ) external verifyRawFulfillment {
        fulfillDataRequestBytes(requestId, response);
    }
}
