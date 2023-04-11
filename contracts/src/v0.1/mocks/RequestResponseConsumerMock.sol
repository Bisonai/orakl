// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerFulfill.sol";
import "../interfaces/RequestResponseCoordinatorInterface.sol";

contract RequestResponseConsumerMock is
    RequestResponseConsumerFulfillUint256,
    RequestResponseConsumerFulfillInt256,
    RequestResponseConsumerFulfillBool,
    RequestResponseConsumerFulfillString,
    RequestResponseConsumerFulfillBytes32,
    RequestResponseConsumerFulfillBytes
{
    using Orakl for Orakl.Request;
    uint256 public sResponse;
    int256 public sResponseInt256;
    bool public sResponseBool;
    string public sResponseString;
    bytes32 public sResponseBytes32;
    bytes public sResponseBytes;

    address private sOwner;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address coordinator) RequestResponseConsumerBase(coordinator) {
        sOwner = msg.sender;
    }

    // Receive remaining payment from requestDataPayment
    receive() external payable {}

    //request for uint256
    function requestDataUint256(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint256"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentUint256(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint256"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    // request for int256
    function requestDataInt256(
        uint64 accId,
        uint32 callbackGasLimit
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

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentInt256(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("int256"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    // request for bool
    function requestDataBool(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentBool(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bool"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    // request for string
    function requestDataString(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentString(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("string"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    // request for bytes32
    function requestDataBytes32(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentBytes32(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes32"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    // request for bytes
    function requestDataBytes(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId);
    }

    function requestDataDirectPaymentBytes(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("bytes"));
        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(req, callbackGasLimit);
    }

    function fulfillDataRequestUint256(uint256 /*requestId*/, uint256 response) internal override {
        sResponse = response;
    }

    function fulfillDataRequestInt256(uint256 /*requestId*/, int256 response) internal override {
        sResponseInt256 = response;
    }

    function fulfillDataRequestBool(uint256 /*requestId*/, bool response) internal override {
        sResponseBool = response;
    }

    function fulfillDataRequestString(
        uint256 /*requestId*/,
        string memory response
    ) internal override {
        sResponseString = response;
    }

    function fulfillDataRequestBytes32(uint256 /*requestId*/, bytes32 response) internal override {
        sResponseBytes32 = response;
    }

    function fulfillDataRequestBytes(
        uint256 /*requestId*/,
        bytes memory response
    ) internal override {
        sResponseBytes = response;
    }
}
