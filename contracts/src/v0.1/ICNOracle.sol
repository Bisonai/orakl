// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import './interfaces/IOracle.sol';

contract ICNOracle is IOracle {
  mapping(uint256 => bool) public s_requestStatuses;

  // Mapping to store results of requests done
  mapping(uint256 => bytes32) public s_requestResults;

  event NewRequest(
    bytes32 indexed requestId,
    bytes32 jobId,
    uint256 nonce,
    address callbackAddress,
    bytes4 callbackFunctionId,
    bytes _data
  );

  function createNewRequest(
    bytes32 _requestId,
    bytes32 _jobId,
    uint256 _nonce,
    address _callbackAddress,
    bytes4 _callbackFunctionId,
    bytes calldata _data
  ) external {
      emit NewRequest(_requestId, _jobId, _nonce, _callbackAddress, _callbackFunctionId, _data);
  }

  function fulfillRequest(bytes32 _requestId, bytes32 _data) external {
    // s_jobResults[_jobId] = data;
    // s_jobStatuses[_jobId] = true;
  }
}
