// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import {InspectorConsumerBase} from "./InspectorConsumerBase.sol";
import {Orakl} from "@bisonai/orakl-contracts/src/v0.1/libraries/Orakl.sol";

contract InspectorConsumer is InspectorConsumerBase{
    using Orakl for Orakl.Request;

    uint256 public sRandomWord;
    uint128 public sResponse;
    address private sOwner;


    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address _rrCoordinator, address _vrfCoordinator)InspectorConsumerBase(_vrfCoordinator, _rrCoordinator){
        sOwner = msg.sender;
    }

    // Receive remaining payment from requestDataPayment
    receive() external payable {}

    function requestRR(
        uint64 accId,
        uint32 callbackGasLimit
    ) public onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        uint8 numSubmission = 1;

        Orakl.Request memory req = buildRequest(jobId);
        req.add("get", "https://api.coinbase.com/v2/exchange-rates?currency=BTC");
        req.add("path", "data,rates,USDT");
        req.add("pow10", "8");

        requestId = rrCoordinator.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    function requestVRF(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public onlyOwner returns (uint256 requestId) {
        requestId = vrfCoordinator.requestRandomWords(keyHash, accId, callbackGasLimit, numWords);
    }

    function fulfillRandomWords(
        uint256 /* requestId */,
        uint256[] memory randomWords
    ) internal override {
        sRandomWord = randomWords[0];
    }

    function fulfillDataRequest(uint256 /*requestId*/, uint128 response) internal override {
        sResponse = response;
    }

    function cancelRequest(uint256 requestId) external onlyOwner {
        rrCoordinator.cancelRequest(requestId);
    }
}
