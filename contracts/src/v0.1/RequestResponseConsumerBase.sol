// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/RequestResponseCoordinatorInterface.sol";

abstract contract RequestResponseConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);
    RequestResponseCoordinatorInterface public immutable COORDINATOR;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        COORDINATOR = RequestResponseCoordinatorInterface(_requestResponseCoordinator);
    }

    /**
     * @notice Build a request using the Orakl library
     * @param jobId the job specification ID that the request is created for
     * @return req request in memory
     */
    function buildRequest(bytes32 jobId) internal view returns (Orakl.Request memory req) {
        bytes4 callbackFunc = detectCallbackFunc(jobId);
        return req.initialize(jobId, address(COORDINATOR), callbackFunc);
    }

    function detectCallbackFunc(bytes32 jobId) internal view returns (bytes4 callbackFunc) {
        if (jobId == keccak256("uint256")) {
            return COORDINATOR.fulfillDataRequestUint256.selector;
        } else if (jobId == keccak256("int256")) {
            return COORDINATOR.fulfillDataRequestInt256.selector;
        } else if (jobId == keccak256("bool")) {
            return COORDINATOR.fulfillDataRequestBool.selector;
        } else if (jobId == keccak256("string")) {
            return COORDINATOR.fulfillDataRequestString.selector;
        } else if (jobId == keccak256("bytes32")) {
            return COORDINATOR.fulfillDataRequestBytes32.selector;
        } else if (jobId == keccak256("bytes")) {
            return COORDINATOR.fulfillDataRequestBytes.selector;
        }
    }

    function fulfillDataRequestUint256(uint256 requestId, uint256 response) internal virtual;

    function fulfillDataRequestInt256(uint256 requestId, int256 response) internal virtual;

    function fulfillDataRequestBool(uint256 requestId, bool response) internal virtual;

    function fulfillDataRequestString(uint256 requestId, string memory response) internal virtual;

    function fulfillDataRequestBytes32(uint256 requestId, bytes32 response) internal virtual;

    function fulfillDataRequestBytes(uint256 requestId, bytes memory response) internal virtual;

    function rawFulfillDataRequestUint256(
        uint256 requestId,
        uint256 response
    ) external verifyRawFulfillment {
        fulfillDataRequestUint256(requestId, response);
    }

    function rawFulfillDataRequestInt256(
        uint256 requestId,
        int256 response
    ) external verifyRawFulfillment {
        fulfillDataRequestInt256(requestId, response);
    }

    function rawFulfillDataRequestBool(
        uint256 requestId,
        bool response
    ) external verifyRawFulfillment {
        fulfillDataRequestBool(requestId, response);
    }

    function rawFulfillDataRequestString(
        uint256 requestId,
        string memory response
    ) external verifyRawFulfillment {
        fulfillDataRequestString(requestId, response);
    }

    function rawFulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response
    ) external verifyRawFulfillment {
        fulfillDataRequestBytes32(requestId, response);
    }

    function rawFulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response
    ) external verifyRawFulfillment {
        fulfillDataRequestBytes(requestId, response);
    }

    modifier verifyRawFulfillment() {
        address coordinatorAddress = address(COORDINATOR);
        if (msg.sender != coordinatorAddress) {
            revert OnlyCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }
        _;
    }
}
