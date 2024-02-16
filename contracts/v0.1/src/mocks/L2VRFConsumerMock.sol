// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../VRFConsumerBase.sol";
import "../interfaces/IL2Endpoint.sol";

contract L2VRFConsumerMock is VRFConsumerBase {
    uint256 public sRandomWord;
    address private sOwner;

    IL2Endpoint L2ENDPOINT;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != sOwner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address l2Endpoint) VRFConsumerBase(l2Endpoint) {
        sOwner = msg.sender;
        L2ENDPOINT = IL2Endpoint(l2Endpoint);
    }

    // Receive remaining payment from requestRandomWordsPayment
    receive() external payable {}

    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public onlyOwner returns (uint256 requestId) {
        requestId = L2ENDPOINT.requestRandomWords(keyHash, accId, callbackGasLimit, numWords);
    }

    function fulfillRandomWords(
        uint256 /* requestId */,
        uint256[] memory randomWords
    ) internal override {
        sRandomWord = (randomWords[0] % 50) + 1;
    }
}
