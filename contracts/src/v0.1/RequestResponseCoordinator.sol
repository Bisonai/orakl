// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.7/Operator.sol

import "./interfaces/IOracle.sol";
import "./interfaces/TypeAndVersionInterface.sol";

error FailedToCallback();
error RequestAlreadyExists();
error IncorrectRequest();

contract RequestResponseCoordinator is IOracle, TypeAndVersionInterface {
    // Mapping requestIds => hashes of requests
    mapping(bytes32 => bytes32) private s_requests;

    event Requested(
        bytes32 indexed requestId,
        bytes32 jobId,
        uint256 nonce,
        address callbackAddress,
        bytes4 callbackFunctionId,
        bytes data
    );
    event Fulfilled(bytes32 indexed requestId);
    event Cancelled(bytes32 indexed requestId);

    function createRequest(
        bytes32 _requestId,
        bytes32 _jobId,
        uint256 _nonce,
        bytes4 _callbackFunctionId,
        bytes calldata _data
    ) external {
        address callbackAddress = msg.sender;

        if (s_requests[_requestId] != 0) {
            revert RequestAlreadyExists();
        }
        s_requests[_requestId] = keccak256(
            abi.encodePacked(_requestId, callbackAddress, _callbackFunctionId)
        );

        emit Requested(_requestId, _jobId, _nonce, callbackAddress, _callbackFunctionId, _data);
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
    ) external {
        // TODO - Add validator node check
        bytes32 paramsHash = keccak256(
            abi.encodePacked(_requestId, _callbackAddress, _callbackFunctionId)
        );
        if (s_requests[_requestId] != paramsHash) {
            revert IncorrectRequest();
        }
        delete s_requests[_requestId];
        (bool success, ) = _callbackAddress.call(
            abi.encodeWithSelector(_callbackFunctionId, _requestId, _data)
        );
        if (!success) {
            revert FailedToCallback();
        }
        emit Fulfilled(_requestId);
    }

    /**
     * @notice Cancelling Oracle Request
     * @param _requestId - ID of the Oracle Request
     */
    function cancelRequest(
        bytes32 _requestId,
        bytes4 _callbackFunctionId
    ) external {
        address callbackAddress = msg.sender;
        bytes32 paramsHash = keccak256(
            abi.encodePacked(_requestId, callbackAddress, _callbackFunctionId)
        );
        if (s_requests[_requestId] != paramsHash) {
            revert IncorrectRequest();
        }
        delete s_requests[_requestId];
        emit Cancelled(_requestId);
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "RequestResponseCoordinator v0.1";
    }
}
