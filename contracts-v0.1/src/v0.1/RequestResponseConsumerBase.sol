// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IRequestResponseCoordinator.sol";

abstract contract RequestResponseConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);

    mapping(bytes32 => bytes4) private sJobIdToFunctionSelector;
    IRequestResponseCoordinator public immutable COORDINATOR;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        COORDINATOR = IRequestResponseCoordinator(_requestResponseCoordinator);

        sJobIdToFunctionSelector[keccak256(abi.encodePacked("uint128"))] = COORDINATOR
            .fulfillDataRequestUint128
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("int256"))] = COORDINATOR
            .fulfillDataRequestInt256
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bool"))] = COORDINATOR
            .fulfillDataRequestBool
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("string"))] = COORDINATOR
            .fulfillDataRequestString
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bytes32"))] = COORDINATOR
            .fulfillDataRequestBytes32
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bytes"))] = COORDINATOR
            .fulfillDataRequestBytes
            .selector;
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
