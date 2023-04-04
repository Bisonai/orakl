// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/RequestResponseCoordinatorInterface.sol";

abstract contract RequestResponseConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);
    RequestResponseCoordinatorInterface public immutable COORDINATOR;
    mapping(bytes32 => bytes4) private sJobIdToFunctionSelector;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        COORDINATOR = RequestResponseCoordinatorInterface(_requestResponseCoordinator);
        sJobIdToFunctionSelector[keccak256("uint256")] = COORDINATOR
            .fulfillDataRequestUint256
            .selector;
        sJobIdToFunctionSelector[keccak256("int256")] = COORDINATOR
            .fulfillDataRequestInt256
            .selector;
        sJobIdToFunctionSelector[keccak256("bool")] = COORDINATOR.fulfillDataRequestInt256.selector;
        sJobIdToFunctionSelector[keccak256("string")] = COORDINATOR
            .fulfillDataRequestString
            .selector;
        sJobIdToFunctionSelector[keccak256("bytes32")] = COORDINATOR
            .fulfillDataRequestBytes32
            .selector;
        sJobIdToFunctionSelector[keccak256("bytes")] = COORDINATOR.fulfillDataRequestBytes.selector;
    }

    /**
     * @notice Build a request using the Orakl library
     * @param jobId the job specification ID that the request is created for
     * @return req request in memory
     */
    function buildRequest(bytes32 jobId) internal view returns (Orakl.Request memory req) {
        return req.initialize(jobId, address(COORDINATOR), sJobIdToFunctionSelector[jobId]);
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
