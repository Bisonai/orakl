// SPDX-License-Identifier: MIT
// Reference - https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/ChainlinkClient.sol

import './ICN.sol';

contract ICNClient {
  using ICN for ICN.Request;

  address private s_oracle;
  uint256 private s_requestCount = 1;

  event Requested(bytes32 indexed id);
  event Fulfilled(bytes32 indexed id);
  event Cancelled(bytes32 indexed id);

  /**
   * @notice Creates a request using the ICN library
   * @param _jobId the job specification ID that the request is created for
   * @param _callbackAddr address to operate the callback
   * @param _callbackFunc function to use for callbacl
   * @return chainlink request in memory
   */
  function buildChainlinkRequest(
    bytes32 _jobId,
    address _callbackAddr,
    bytes4 _callbackFunc
  ) internal pure returns (ICN.Request memory) {
    ICN.Request memory req;
    return req.initialize(_jobId, _callbackAddr, _callbackFunc);
  }

  /**
   * @notice Creates a request to the oracle address
   * @dev calls request to stored oracle address
   * @param _req the initialized chainlink request
   * @return requestId the request Id
   */
  function sendRequest(ICN.Request memory _req) internal returns (bytes32) {
    return sendRequestTo(address(s_oracle), _req);
  }

  //
}
