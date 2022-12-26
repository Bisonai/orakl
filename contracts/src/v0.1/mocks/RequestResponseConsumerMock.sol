// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerBase.sol";

contract RequestResponseConsumerMock is RequestResponseConsumerBase {
    using ICN for ICN.Request;

    bytes32 private jobId;
    int256 public value;

    constructor(address _oracleAddress) {
        setOracle(_oracleAddress);
        jobId = keccak256(abi.encodePacked("any-api-int256"));
    }

    function requestData() public returns (bytes32 requestId) {
        ICN.Request memory req = buildRequest(jobId, address(this), this.fulfill.selector);
        req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD");
        req.add("path", "RAW,ETH,USD,PRICE");
        return sendRequest(req);
    }

    function fulfill(bytes32 _requestId, int256 _response) public ICNResponseFulfilled(_requestId) {
        value = _response;
    }
}
