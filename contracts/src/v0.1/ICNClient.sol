// SPDX-License-Identifier: MIT
// Reference - https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/ChainlinkClient.sol
pragma solidity ^0.8.16;

import "./libraries/ICN.sol";
import "./interfaces/IOracle.sol";

error SenderIsNotOracle();

contract ICNClient {
    using ICN for ICN.Request;

    address private s_oracle;
    uint256 private s_requestCount = 1;
    mapping(bytes32 => address) private s_pendingRequests;

    event Requested(bytes32 indexed id);
    event Fulfilled(bytes32 indexed id); // FIXME not used
    event Cancelled(bytes32 indexed id); // FIXME not used

    /**
     * @notice a modifier to declare the request is fulfiled by the oracle
     */
    modifier ICNResponseFulfilled(bytes32 _requestId) {
        if (msg.sender != s_pendingRequests[_requestId]) {
            revert SenderIsNotOracle();
        }
        delete s_pendingRequests[_requestId]; // Gas refund for clearing memory

        _;
        emit Fulfilled(_requestId);
    }

    /**
     * @notice Creates a request using the ICN library
     * @param _jobId the job specification ID that the request is created for
     * @param _callbackAddr address to operate the callback
     * @param _callbackFunc function to use for callbacl
     * @return  req request in memory
     */
    function buildRequest(bytes32 _jobId, address _callbackAddr, bytes4 _callbackFunc)
        internal
        pure
        returns (ICN.Request memory req)
    {
        return req.initialize(_jobId, _callbackAddr, _callbackFunc);
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
    function sendRequestTo(address _oracleAddress, ICN.Request memory _req) internal returns (bytes32 requestId) {
        uint256 nonce = s_requestCount;
        s_requestCount = nonce + 1;
        requestId = keccak256(abi.encodePacked(this, s_requestCount));
        s_pendingRequests[requestId] = _oracleAddress;
        IOracle(_oracleAddress).createNewRequest(
            requestId, _req.id, nonce, address(this), _req.callbackFunctionId, _req.buf.buf
        );

        emit Requested(requestId);
    }

    /**
     * @notice a function to set oracle address
     */
    function setOracle(address _oracleAddress) internal {
        s_oracle = _oracleAddress;
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure returns (string memory) {
        return "ICNClient v0.1";
    }
}
