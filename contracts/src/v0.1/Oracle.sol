// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

contract ICNOracle {
  // Mapping to store completion statuses for requests to verify
  mapping(uint256 => bool) public jobStatuses;

  // Mapping to store results of requests done
  mapping(uint256 => bytes) public jobResults;

  mapping(string => uint256) public urlJobIds;

  uint256 jobId;

  event NewJob(uint256 jobId, string url);

  function fetchData(string calldata url) external {
    urlJobIds[url] = jobId;
    emit NewJob(jobId, url);
    jobId++;
  }

  function setData(bytes calldata data, uint256 jobId) external {
    jobResults[jobId] = data;
    jobStatuses[jobId] = true;
  }

  function getData(string calldata url) external view returns (bytes memory) {
    return jobResults[urlJobIds[url]];
  }
}
