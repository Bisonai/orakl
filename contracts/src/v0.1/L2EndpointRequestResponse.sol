// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./L2EndpointBase.sol";
import "./libraries/Orakl.sol";
import "./RequestResponseConsumerFulfill.sol";

abstract contract L2EndpointRequestResponse is L2EndpointBase {
    using Orakl for Orakl.Request;
    uint64 private sNonce;

    event DataRequested(
        uint256 indexed requestId,
        bytes32 jobId,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        address indexed sender,
        uint8 numSubmission,
        Orakl.Request req
    );

    event DataRequestFulfilledUint128(uint256 indexed requestId, uint256 response, bool success);
    event DataRequestFulfilledInt256(uint256 indexed requestId, int256 response, bool success);
    event DataRequestFulfilledBool(uint256 indexed requestId, bool response, bool success);
    event DataRequestFulfilledString(uint256 indexed requestId, string response, bool success);
    event DataRequestFulfilledBytes32(uint256 indexed requestId, bytes32 response, bool success);
    event DataRequestFulfilledBytes(uint256 indexed requestId, bytes response, bool success);

    function computeRequestId(
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256) {
        return uint256(keccak256(abi.encode(sender, accId, nonce)));
    }

    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numSubmission
    ) external nonReentrant returns (uint256) {
        sNonce++;
        uint256 requestId = computeRequestId(msg.sender, accId, sNonce);
        sRequestDetail[requestId] = RequestInfo({
            owner: msg.sender,
            callbackGasLimit: callbackGasLimit
        });
        emit DataRequested(
            requestId,
            req.id,
            accId,
            callbackGasLimit,
            msg.sender,
            numSubmission,
            req
        );

        return requestId;
    }

    function fulfillDataRequestUint128(uint256 requestId, uint128 response) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillUint128.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledUint128(requestId, response, success);
    }

    function fulfillDataRequestInt256(uint256 requestId, int256 response) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillInt256.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledInt256(requestId, response, success);
    }

    function fulfillDataRequestBool(uint256 requestId, bool response) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBool.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledBool(requestId, response, success);
    }

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response
    ) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillString.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledString(requestId, response, success);
    }

    function fulfillDataRequestBytes32(uint256 requestId, bytes32 response) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes32.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledBytes32(requestId, response, success);
    }

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response
    ) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            RequestResponseConsumerFulfillBytes.rawFulfillDataRequest.selector,
            requestId,
            response
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit DataRequestFulfilledBytes(requestId, response, success);
    }
}
