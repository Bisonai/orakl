// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import './interfaces/IOracle.sol';

contract ICNOracleNew is IOracle {
  // Mapping to store completion statuses for requests to verify

  mapping(uint256 => bool) public s_requestStatuses;

  // Mapping to store results of requests done
  mapping(uint256 => bytes32) public s_requestResults;

  event NewRequest(bytes32 indexed requestId, uint256 nonce, bytes _data);

  function createNewRequest(bytes32 _requestId, uint256 _nonce, bytes calldata _data) external {
    emit NewRequest(_requestId, _nonce, _data);
  }

  function fulfillRequest(bytes32 _requestId, bytes32 _data) external {
    // s_jobResults[_jobId] = data;
    // s_jobStatuses[_jobId] = true;
  }
}
