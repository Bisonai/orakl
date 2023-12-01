// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerBase.sol";
import "../L1EndpointBase.sol";
import "../L1EndpointRequestResponse.sol";
import "../interfaces/IL1Endpoint.sol";

contract L1EndpointConsumerMock is
    RequestResponseConsumerFulfillUint128,
    RequestResponseConsumerFulfillInt256,
    RequestResponseConsumerFulfillBool,
    RequestResponseConsumerFulfillString,
    RequestResponseConsumerFulfillBytes32,
    RequestResponseConsumerFulfillBytes
{
    using Orakl for Orakl.Request;
    uint128 public sResponseUint128;
    int256 public sResponseInt256;
    bool public sResponseBool;
    string public sResponseString;
    bytes32 public sResponseBytes32;
    bytes public sResponseBytes;
    address private sOwner;

    IL1Endpoint L1ENDPOINT;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address l1Endpoint) RequestResponseConsumerBase(l1Endpoint) {
        sOwner = msg.sender;
        L1ENDPOINT = IL1Endpoint(l1Endpoint);
    }

    // Receive remaining payment from requestRandomWordsPayment
    receive() external payable {}

    //request for uint128
    function requestDataUint128(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");
        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    // request for int256
    function requestDataInt256(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("int256"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    // request for bool
    function requestDataBool(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    // request for string
    function requestDataString(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    // request for bytes32
    function requestDataBytes32(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    // request for bytes
    function requestDataBytes(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        uint256 l2RequestId
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L1ENDPOINT.requestData(
            accId,
            callbackGasLimit,
            numSubmission,
            address(this),
            l2RequestId,
            req
        );
    }

    function fulfillDataRequest(uint256 /*requestId*/, uint128 response) internal override {
        sResponseUint128 = response;
    }

    function fulfillDataRequest(uint256 /*requestId*/, int256 response) internal override {
        sResponseInt256 = response;
    }

    function fulfillDataRequest(uint256 /*requestId*/, bool response) internal override {
        sResponseBool = response;
    }

    function fulfillDataRequest(uint256 /*requestId*/, string memory response) internal override {
        sResponseString = response;
    }

    function fulfillDataRequest(uint256 /*requestId*/, bytes32 response) internal override {
        sResponseBytes32 = response;
    }

    function fulfillDataRequest(uint256 /*requestId*/, bytes memory response) internal override {
        sResponseBytes = response;
    }
}
