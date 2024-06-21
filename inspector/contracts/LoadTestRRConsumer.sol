// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import {InspectorConsumerBase} from "./InspectorConsumerBase.sol";
import {Orakl} from "@bisonai/orakl-contracts/src/v0.1/libraries/Orakl.sol";

contract LoadTestRRConsumer is InspectorConsumerBase{
    using Orakl for Orakl.Request;

    uint256[] public blockRecords;
    mapping(uint256 => uint256) public requestBlockNumbers;
    uint256[] public allRequestIds;
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
    ) public onlyOwner {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        uint8 numSubmission = 1;

        Orakl.Request memory req = buildRequest(jobId);
        req.add("get", "https://api.coinbase.com/v2/exchange-rates?currency=BTC");
        req.add("path", "data,rates,USDT");
        req.add("pow10", "8");

        uint256 requestId;
        for (uint16 i = 0; i < 50; i++) {
            requestId = rrCoordinator.requestData(req, callbackGasLimit, accId, numSubmission);
            requestBlockNumbers[requestId] = block.number;
            allRequestIds.push(requestId);
        }
    }

    function fulfillDataRequest(uint256 requestId, uint128 /* response */) internal override {
        blockRecords.push(block.number - requestBlockNumbers[requestId]);
    }

    function fulfillRandomWords(
        uint256 /* requestId */,
        uint256[] memory  /* randomWords */
    ) internal override {
        // pass
    }

    function getBlockRecordsLength() public view returns (uint256) {
        return blockRecords.length;
    }

    function clear() public onlyOwner {
        for (uint256 i = 0; i < allRequestIds.length; i++) {
            delete requestBlockNumbers[allRequestIds[i]];
        }

        delete blockRecords;
        delete allRequestIds;
    }
}
