// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

contract ICNOracle {
  // Mapping to store completion statuses for requests to verify
  mapping(uint256 => bool) public jobStatuses;

  // Mapping to store results of requests done
  mapping(uint256 => bytes32) public jobResults;

  uint256 jobId;

  event NewJob(uint256 jobId, string url);

  function createNewJob(string calldata url) external {
    emit NewJob(jobId, url);
    jobId++;
  }

  function fulfillJob(bytes32 data, uint256 jobId) external {
    jobResults[jobId] = data;
    jobStatuses[jobId] = true;
  }

  function getData(uint256 jobId) external view returns (bytes32) {
    return jobResults[jobId];
  }
}