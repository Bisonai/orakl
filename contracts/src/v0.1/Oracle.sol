// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

contract ICNOracle {
  // Mapping to store completion statuses for requests to verify
  mapping(uint256 => bool) public s_jobStatuses;

  // Mapping to store results of requests done
  mapping(uint256 => bytes32) public s_jobResults;

  uint256 s_jobId;

  event NewJob(uint256 jobId, string url);

  function createNewJob(string calldata url) external {
    emit NewJob(s_jobId, url);
    s_jobId++;
  }

  function fulfillJob(bytes32 data, uint256 jobId) external {
    s_jobResults[jobId] = data;
    s_jobStatuses[jobId] = true;
  }

  function getData(uint256 jobId) external view returns (bytes32) {
    return s_jobResults[jobId];
  }
}
