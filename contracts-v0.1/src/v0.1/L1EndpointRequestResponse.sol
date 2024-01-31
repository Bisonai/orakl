// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./RequestResponseConsumerBase.sol";
import "./L1EndpointBase.sol";
import "./RequestResponseConsumerFulfill.sol";

abstract contract L1EndpointRequestResponse is
    L1EndpointBase,
    RequestResponseConsumerFulfillUint128,
    RequestResponseConsumerFulfillInt256,
    RequestResponseConsumerFulfillBool,
    RequestResponseConsumerFulfillString,
    RequestResponseConsumerFulfillBytes32,
    RequestResponseConsumerFulfillBytes
{
    using Orakl for Orakl.Request;

    event DataRequested(uint256 requestId, address sender);
    event DataRequestFulfilled(
        uint256 requestId,
        uint256 l2RequestId,
        address sender,
        uint256 callbackGasLimit,
        bytes32 jobId,
        uint128 responseUint128,
        int256 responseInt256,
        bool responseBool,
        string responseString,
        bytes32 responseBytes32,
        bytes responseBytes
    );

    constructor(
        address requestResponseCoordinator
    ) RequestResponseConsumerBase(requestResponseCoordinator) {}

    function requestData(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId,
        Orakl.Request memory req
    ) public returns (uint256) {
        uint64 reqCount = 0;
        uint256 fee = COORDINATOR.estimateFee(reqCount, 1, callbackGasLimit);
        pay(accId, sender, fee);
        uint256 id = COORDINATOR.requestData{value: fee}(
            req,
            callbackGasLimit,
            numSubmission,
            address(this)
        );
        sRequest[id] = RequestDetail(l2RequestId, sender, callbackGasLimit);
        emit DataRequested(id, sender);
        return id;
    }

    function fulfillDataRequest(uint256 requestId, uint128 response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            response,
            0,
            false,
            "",
            "",
            ""
        );
        delete sRequest[requestId];
    }

    function fulfillDataRequest(uint256 requestId, int256 response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("int256"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            0,
            response,
            false,
            "",
            "",
            ""
        );
        delete sRequest[requestId];
    }

    function fulfillDataRequest(uint256 requestId, bool response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            0,
            0,
            response,
            "",
            "",
            ""
        );
        delete sRequest[requestId];
    }

    function fulfillDataRequest(uint256 requestId, string memory response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            0,
            0,
            false,
            response,
            "",
            ""
        );
        delete sRequest[requestId];
    }

    function fulfillDataRequest(uint256 requestId, bytes32 response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            0,
            0,
            false,
            "",
            response,
            ""
        );
        delete sRequest[requestId];
    }

    function fulfillDataRequest(uint256 requestId, bytes memory response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        RequestDetail memory r = sRequest[requestId];
        emit DataRequestFulfilled(
            requestId,
            r.l2RequestId,
            r.sender,
            r.callbackGasLimit,
            jobId,
            0,
            0,
            false,
            "",
            "",
            response
        );
        delete sRequest[requestId];
    }
}
