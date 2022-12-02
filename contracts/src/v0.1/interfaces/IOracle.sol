// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IOracle {
  /**
   * @notice Function to create a new oracle request
   */
  function createNewRequest(
    bytes32 _requestId,
    bytes32 _jobId,
    uint256 _nonce,
    address _callbackAddress,
    bytes4 _callbackFunctionId,
    bytes calldata _data
  ) external;
}
