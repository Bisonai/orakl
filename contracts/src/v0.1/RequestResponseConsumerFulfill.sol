// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./RequestResponseConsumerBase.sol";

abstract contract RequestResponseConsumerFulfillUint128 is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, uint128 response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        uint128 response
    ) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillInt256 is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, int256 response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        int256 response
    ) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBool is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, bool response) internal virtual;

    function rawFulfillDataRequest(uint256 requestId, bool response) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillString is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, string memory response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        string memory response
    ) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBytes32 is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, bytes32 response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        bytes32 response
    ) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}

abstract contract RequestResponseConsumerFulfillBytes is RequestResponseConsumerBase {
    function fulfillDataRequest(uint256 requestId, bytes memory response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        bytes memory response
    ) external verifyRawFulfillment {
        fulfillDataRequest(requestId, response);
    }
}
