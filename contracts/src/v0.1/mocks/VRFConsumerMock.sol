// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import '../VRFConsumerBase.sol';
import '../interfaces/VRFCoordinatorInterface.sol';


contract VRFConsumerMock is VRFConsumerBase {
  uint256 public s_randomResult;
  address private s_owner;

  VRFCoordinatorInterface COORDINATOR;

  error OnlyOwner(address notOwner);

  modifier onlyOwner() {
      if (msg.sender != s_owner) {
          revert OnlyOwner(msg.sender);
      }
      _;
  }

  constructor(address coordinator)
      VRFConsumerBase(coordinator)
      // ConfirmedOwner(msg.sender) TODO
  {
      s_owner = msg.sender;
      COORDINATOR = VRFCoordinatorInterface(coordinator);
  }

  function requestRandomWords( bytes32 keyHash, uint64 accId,uint16 requestConfirmations,uint32 callbackGasLimit,uint32 numWords ) public returns(uint256 requestId) {

    requestId = COORDINATOR.requestRandomWords(
      keyHash,
      accId,
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

  function fulfillRandomWords(uint256 /* requestId */, uint256[] memory randomWords) internal override {
    // requestId should be checked if it matches the expected request
    s_randomResult = (randomWords[0] % 50) + 1;
  }

  function requestRandomWordsDirect(
    ) public payable returns (uint256 requestId) {

    uint16 requestConfirmations = 3;
    uint32 callbackGasLimit = 1_000_000;
    uint32 numWords = 1;


    requestId = COORDINATOR.requestRandomWordsPayment{value:msg.value}(
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }
}
