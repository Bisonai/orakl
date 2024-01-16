// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import {Orakl} from "@bisonai/orakl-contracts/src/v0.1/libraries/Orakl.sol";
import {IRequestResponseCoordinator} from "@bisonai/orakl-contracts/src/v0.1/interfaces/IRequestResponseCoordinator.sol";
import {IVRFCoordinator} from "@bisonai/orakl-contracts/src/v0.1/interfaces/IVRFCoordinator.sol";

abstract contract InspectorConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);

    IVRFCoordinator public immutable vrfCoordinator;
    IRequestResponseCoordinator public immutable rrCoordinator;
    uint256 public vrfRequestId;
    uint256 public rrRequestId;
    mapping(bytes32 => bytes4) private sJobIdToFunctionSelector;

    constructor(address _vrfCoordinator, address _rrCoordinator) {
        vrfCoordinator = IVRFCoordinator(_vrfCoordinator);
        rrCoordinator = IRequestResponseCoordinator(_rrCoordinator);
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("uint128"))] = rrCoordinator
            .fulfillDataRequestUint128
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("int256"))] = rrCoordinator
            .fulfillDataRequestInt256
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bool"))] = rrCoordinator
            .fulfillDataRequestBool
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("string"))] = rrCoordinator
            .fulfillDataRequestString
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bytes32"))] = rrCoordinator
            .fulfillDataRequestBytes32
            .selector;
        sJobIdToFunctionSelector[keccak256(abi.encodePacked("bytes"))] = rrCoordinator
            .fulfillDataRequestBytes
            .selector;
    }

    function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) internal virtual;

    function rawFulfillRandomWords(uint256 requestId, uint256[] memory randomWords) external {
        vrfRequestId = requestId;
        address coordinatorAddress = address(vrfCoordinator);
        if (msg.sender != coordinatorAddress) {
            revert OnlyCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }
        fulfillRandomWords(requestId, randomWords);
    }

    function buildRequest(bytes32 jobId) internal view returns (Orakl.Request memory req) {
        return req.initialize(jobId, address(rrCoordinator), sJobIdToFunctionSelector[jobId]);
    }

    function fulfillDataRequest(uint256 requestId, uint128 response) internal virtual;

    function rawFulfillDataRequest(
        uint256 requestId,
        uint128 response
    ) external {
        rrRequestId = requestId;
        address coordinatorAddress = address(rrCoordinator);
        if (msg.sender != coordinatorAddress) {
            revert OnlyCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }

        fulfillDataRequest(requestId, response);
    }
}
