// SPDX-License-Identifier: MIT
// Reference - https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/Chainlink.sol
pragma solidity ^0.8.16;

import {Buffer} from './libraries/Buffer.sol';
import {CBOR} from './libraries/CBOR.sol';

/// @title ICN Library
/// @author Zahid Ahmed
/// @notice ICN Library Contract for common functions

library ICN {
  using CBOR for Buffer.buffer;

  // structure for storing requests done off-chain
  struct Request {
    bytes32 id;
    address callbackAddress;
    bytes4 callbackFunctionId;
    uint256 nonce;
    Buffer.buffer buf;
  }

  // Declaring default buffer size
  uint256 internal constant defaultBuffSize = 256;

  /**
   * @notice Initializes a request
   * @dev Sets ID, callback address, and callback function
   * @param _request The uninitialized request
   * @param _jobId The Job Specification ID
   * @param _callbackAddr The callback address
   * @param _callbackFunc The callback function signature
   * @return The initialized request
   */
  function initialize(
    Request memory _request,
    bytes32 _jobId,
    address _callbackAddr,
    bytes4 _callbackFunc
  ) internal pure returns (ICN.Request memory) {
    Buffer.init(_request.buf, defaultBuffSize);
    _request.id = _jobId;
    _request.callbackAddress = _callbackAddr;
    _request.callbackFunctionId = _callbackFunc;
    return _request;
  }

  /**
   * @notice sets the data for buffer
   * @param _request the initialized request
   * @param _data the CBOR data
   */
  function setBuffer(Request memory _request, bytes memory _data) internal pure {
    Buffer.init(_request.buf, _data.length);
    Buffer.append(_request.buf, _data);
  }
}
