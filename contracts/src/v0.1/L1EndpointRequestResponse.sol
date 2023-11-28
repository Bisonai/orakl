// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./RequestResponseConsumerBase.sol";
import "./L1EndpointBase.sol";
import "./RequestResponseConsumerFulfill.sol";

abstract contract L1EndpointRequestResponse is
    RequestResponseConsumerBase,
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
    ) internal returns (uint256) {
        if (!sOracles[msg.sender]) {
            revert OnlyOracle();
        }
        //check consumer and balance
        bool isValidConsumer = REGISTRY.isValidConsumer(accId, sender);
        if (!isValidConsumer) {
            revert ConsumerValid();
        }
        uint256 balance = REGISTRY.getBalance(accId);
        uint64 reqCount = 0;
        uint256 fee = COORDINATOR.estimateFee(reqCount, 1, callbackGasLimit);
        REGISTRY.decreaseBalance(accId, fee);
        if (balance < fee) {
            revert InsufficientBalance();
        }
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

    //request data
    function requestDataUint128(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    // request for int256
    function requestDataInt256(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("int256"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    // request for bool
    function requestDataBool(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    // request for string
    function requestDataString(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    // request for bytes32
    function requestDataBytes32(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    // request for bytes
    function requestDataBytes(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = requestData(accId, callbackGasLimit, numSubmission, sender, l2RequestId, req);
    }

    function fulfillDataRequest(uint256 requestId, uint128 response) internal override {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
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
