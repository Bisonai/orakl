// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import '../VRFConsumerBase.sol';
import '../interfaces/VRFCoordinatorInterface.sol';


contract VRFConsumerMock is VRFConsumerBase {
  uint256 public randomResult;

  VRFCoordinatorInterface COORDINATOR;

  constructor(address coordinator)
      VRFConsumerBase(coordinator)
      // ConfirmedOwner(msg.sender) TODO
  {
      COORDINATOR = VRFCoordinatorInterface(coordinator);
  }

  function request() public {
    bytes32 keyHash;
    uint64 subId;
    uint16 requestConfirmations;
    uint32 callbackGasLimit;
    uint32 numWords;

    uint256 requestId = COORDINATOR.requestRandomWords(
      keyHash,
      subId,
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

  function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) internal override {
    // requestId should be checked if it matches the expected request
    randomResult = (randomWords[0] % 50) + 1;
  }
}
