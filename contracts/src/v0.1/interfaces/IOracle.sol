// SPDX-License-Identifier: MIT

interface IOracle {
  /**
   * @notice Function to create a new oracle request
   */
  function createNewRequest(bytes32 _jobId, uint256 _nonce, bytes calldata _data) external;
}
