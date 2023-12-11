// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerFulfill.sol";
import "../RequestResponseConsumerBase.sol";
import "../interfaces/IL2Endpoint.sol";

contract L2RequestResponseConsumerMock is
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

    IL2Endpoint L2ENDPOINT;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address l2Endpoint) RequestResponseConsumerBase(l2Endpoint) {
        sOwner = msg.sender;
        L2ENDPOINT = IL2Endpoint(l2Endpoint);
    }

    // Receive remaining payment from requestRandomWordsPayment
    receive() external payable {}

    //request for uint128
    function requestDataUint128(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
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
        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    // request for int256
    function requestDataInt256(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
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

        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    // request for bool
    function requestDataBool(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add("get", "https://mockapi.impetus.fun/true");
        req.add("path", "data");

        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    // request for string
    function requestDataString(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    // request for bytes32
    function requestDataBytes32(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add(
            "get",
            "https://mockapi.impetus.fun/0x68656c6c6f000000000000000000000000000000000000000000000000000000"
        );
        req.add("path", "data");
        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    // request for bytes
    function requestDataBytes(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        Orakl.Request memory req = buildRequest(jobId);
        req.add("get", "https://mockapi.impetus.fun/0x68656c6c6f");
        req.add("path", "data");

        requestId = L2ENDPOINT.requestData(req, callbackGasLimit, accId, numSubmission);
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
