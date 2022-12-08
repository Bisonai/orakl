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

  function requestRandomWords() public returns(uint256 requestId) {
    bytes32 keyHash;
    uint64 subId = 1;
    uint16 requestConfirmations = 3;
    uint32 callbackGasLimit = 3;
    uint32 numWords = 1;

    requestId = COORDINATOR.requestRandomWords(
      keyHash,
      subId,
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

  function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) internal override {
    // requestId should be checked if it matches the expected request
    s_randomResult = (randomWords[0] % 50) + 1;
  }
}
