// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import '../ICNClient.sol';

contract ICNMock is ICNClient {
  using ICN for ICN.Request;

  bytes32 private jobId;

  constructor(address _oracleAddress) {
    setOracle(_oracleAddress);
    // FIXME change jobId
    jobId = 'TEST1';
  }

  function requestData() public returns (bytes32 requestId) {
    ICN.Request memory req = buildRequest(jobId, address(this), this.fulfill.selector);

    req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD");
    requestId = sendRequest(req);
  }

  function fulfill(bytes32 _requestId, bytes memory data) public {}
}
