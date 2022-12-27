// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/ChainlinkClient.sol

import "./libraries/ICN.sol";
import "./interfaces/IOracle.sol";

error SenderIsNotOracle();

abstract contract RequestResponseConsumerBase {
    using ICN for ICN.Request;

    address private s_oracle;
    uint256 private s_requestCount = 1;
    mapping(bytes32 => address) private s_pendingRequests;

    /**
     * @notice a modifier to declare the request is fulfiled by the oracle
     */
    modifier ICNResponseFulfilled(bytes32 _requestId) {
        if (msg.sender != s_pendingRequests[_requestId]) {
            revert SenderIsNotOracle();
        }
        delete s_pendingRequests[_requestId];

        _;
    }

    /**
     * @notice Set oracle address
     */
    function setOracle(address _oracleAddress) internal {
        s_oracle = _oracleAddress;
    }

    /**
     * @notice Creates a request using the ICN library
     * @param _jobId the job specification ID that the request is created for
     * @param _callbackAddr address to operate the callback
     * @param _callbackFunctionId function to use for callback
     * @return  req request in memory
     */
    function buildRequest(
        bytes32 _jobId,
        address _callbackAddr,
        bytes4 _callbackFunctionId
    ) internal pure returns (ICN.Request memory req) {
        return req.initialize(_jobId, _callbackAddr, _callbackFunctionId);
    }

    /**
     * @notice Creates a request to the oracle address
     * @dev calls request to stored oracle address
     * @param _req the initialized  request
     * @return requestId the request Id
     */
    function sendRequest(ICN.Request memory _req) internal returns (bytes32) {
        return sendRequestTo(address(s_oracle), _req);
    }

    /**
     * @notice Creates a request to the oracle address
     * @dev Generates and stores a request ID, increments the local nonce, creates a request on the target oracle contract.
     * Emits Requested event.
     * @param _oracleAddress The address of the oracle for the request
     * @param _req The initialized Request
     * @return requestId The request ID
     */
    function sendRequestTo(
        address _oracleAddress,
        ICN.Request memory _req
    ) internal returns (bytes32 requestId) {
        uint256 nonce = s_requestCount;
        s_requestCount = nonce + 1;
        requestId = keccak256(abi.encodePacked(this, s_requestCount));
        s_pendingRequests[requestId] = _oracleAddress;
        IOracle(_oracleAddress).createRequest(
            requestId,
            _req.id,
            nonce,
            _req.callbackFunctionId,
            _req.buf.buf
        );
    }

    /**
     * @notice Cancel request
     * @param _requestId - ID of the Oracle Request
     * @param _callbackFunctionId - Return functionID callback
     */
    function cancelRequest(bytes32 _requestId, bytes4 _callbackFunctionId) internal {
        IOracle(s_oracle).cancelRequest(_requestId, _callbackFunctionId);
        delete s_pendingRequests[_requestId];
    }
}
