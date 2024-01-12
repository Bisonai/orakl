// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import {IAggregatorRouter} from "@bisonai/orakl-contracts/src/v0.1/interfaces/IAggregatorRouter.sol";
import {RequestResponseConsumerFulfillUint128} from "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerFulfill.sol";
import {RequestResponseConsumerBase} from "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerBase.sol";
import {Orakl} from "@bisonai/orakl-contracts/src/v0.1/libraries/Orakl.sol";
import {IPrepayment} from "@bisonai/orakl-contracts/src/v0.1/interfaces/IPrepayment.sol";
import {VRFConsumerBase} from "@bisonai/orakl-contracts/src/v0.1/VRFConsumerBase.sol";
import {IVRFCoordinator} from "@bisonai/orakl-contracts/src/v0.1/interfaces/IVRFCoordinator.sol";

contract InspectorConsumer is RequestResponseConsumerFulfillUint128, VRFConsumerBase{
    using Orakl for Orakl.Request;
    uint256 public sRandomWord;
    uint128 public sResponse;
    address private sOwner;

    IVRFCoordinator vrfCoordinator;
    IAggregatorRouter internal router;
    int256 public dataFeedAnswer;
    uint80 public dataFeedRoundId;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address aggregatorRouter, address rrCoordinator, address _vrfCoordinator) RequestResponseConsumerBase(rrCoordinator) VRFConsumerBase(_vrfCoordinator){
        sOwner = msg.sender;
        router = IAggregatorRouter(aggregatorRouter);
        vrfCoordinator = IVRFCoordinator(_vrfCoordinator);
    }

    // Receive remaining payment from requestDataPayment
    receive() external payable {}


    function requestDataFeed(string calldata pair) public {
        (
            uint80 roundId_,
            int256 answer_
            , /* uint startedAt */
            , /* uint updatedAt */
            , /* uint80 answeredInRound */
        ) = router.latestRoundData(pair);

        dataFeedAnswer = answer_;
        dataFeedRoundId = roundId_;
    }

    function decimals(string calldata pair) public view returns (uint8) {
        return router.decimals(pair);
    }

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

        requestId = COORDINATOR.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    function requestRRDirect(
        uint32 callbackGasLimit
    ) public payable onlyOwner returns (uint256 requestId) {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        uint8 numSubmission = 1;

        Orakl.Request memory req = buildRequest(jobId);
        req.add("get", "https://api.coinbase.com/v2/exchange-rates?currency=BTC");
        req.add("path", "data,rates,USDT");
        req.add("pow10", "8");

        requestId = COORDINATOR.requestData{value: msg.value}(
            req,
            callbackGasLimit,
            numSubmission,
            address(this)
        );
    }

    function requestVRF(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public onlyOwner returns (uint256 requestId) {
        requestId = vrfCoordinator.requestRandomWords(keyHash, accId, callbackGasLimit, numWords);
    }

    function requestVRFDirect(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        address refundRecipient
    ) public payable onlyOwner returns (uint256 requestId) {
        requestId = vrfCoordinator.requestRandomWords{value: msg.value}(
            keyHash,
            callbackGasLimit,
            numWords,
            refundRecipient
        );
    }

    function fulfillRandomWords(
        uint256 /* requestId */,
        uint256[] memory randomWords
    ) internal override {
        // requestId should be checked if it matches the expected request
        // Generate random value between 1 and 50.
        sRandomWord = (randomWords[0] % 50) + 1;
    }

    function fulfillDataRequest(uint256 /*requestId*/, uint128 response) internal override {
        sResponse = response;
    }

    function cancelRequest(uint256 requestId) external onlyOwner {
        COORDINATOR.cancelRequest(requestId);
    }


}