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

    modifier verifyRawFulfillment() {
        address coordinatorAddress = address(COORDINATOR);
        if (msg.sender != coordinatorAddress) {
            revert OnlyCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }
        _;
    }
}
