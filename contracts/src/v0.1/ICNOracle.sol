// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import './interfaces/IOracle.sol';

error RequestAlreadyExists();
error IncorrectRequestFulfillment();

contract ICNOracle is IOracle {
  // Mapping RequestIDs => Hashes of Requests Data
  mapping(bytes32 => bytes32) private requests;

  event NewRequest(
    bytes32 indexed requestId,
    bytes32 jobId,
    uint256 nonce,
    address callbackAddress,
    bytes4 callbackFunctionId,
    bytes _data
  );

  function createNewRequest(
    bytes32 _requestId,
    bytes32 _jobId,
    uint256 _nonce,
    address _callbackAddress,
    bytes4 _callbackFunctionId,
    bytes calldata _data
  ) external {
    if (requests[_requestId] != 0) {
      revert RequestAlreadyExists();
    }
    requests[_requestId] = keccak256(
      abi.encodePacked(_requestId, _callbackAddress, _callbackFunctionId)
    );

    emit NewRequest(_requestId, _jobId, _nonce, _callbackAddress, _callbackFunctionId, _data);
  }

  /**
   * @notice Fulfils oracle request
   * @param _requestId - ID of the Oracle Request
   * @param _callbackAddress - Callback Address of Oracle Fulfilment
   * @param _callbackFunctionId - Return functionID callback
   * @param _data - Return data for fulfilment
   */
  function fulfillOracleRequest(
    bytes32 _requestId,
    address _callbackAddress,
    bytes4 _callbackFunctionId,
    bytes calldata _data
  ) external returns (bool) {
    bytes32 paramsHash = keccak256(
      abi.encodePacked(_requestId, _callbackAddress, _callbackFunctionId)
    );
    if (requests[_requestId] != paramsHash) {
      revert IncorrectRequestFulfillment();
    }
    delete requests[_requestId];
    (bool success, ) = _callbackAddress.call(
      abi.encodeWithSelector(_callbackFunctionId, _requestId, _data)
    );
    return success;
  }

  /**
   * @notice The type and version of this contract
   * @return Type and version string
   */
  function typeAndVersion() external pure returns (string memory) {
    return 'ICNClient v0.1';
  }
}
