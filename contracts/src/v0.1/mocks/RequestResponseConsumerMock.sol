// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerBase.sol";

contract RequestResponseConsumerMock is RequestResponseConsumerBase {
    using ICN for ICN.Request;

    bytes32 private s_jobId;
    int256 public s_response;

    constructor(address _oracleAddress) {
        setOracle(_oracleAddress);
        s_jobId = keccak256(abi.encodePacked("any-api-int256"));
    }

    function makeRequest() public returns (bytes32 requestId) {
        ICN.Request memory req = buildRequest(s_jobId, address(this), this.fulfillRequest.selector);
        req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD");
        req.add("path", "RAW,ETH,USD,PRICE");
        return sendRequest(req);
    }

    function cancelRequest(bytes32 _requestId) public {
        cancelRequest(_requestId, this.fulfillRequest.selector);
    }

    function fulfillRequest(bytes32 _requestId, int256 _response) public ICNResponseFulfilled(_requestId) {
        s_response = _response;
    }
}
