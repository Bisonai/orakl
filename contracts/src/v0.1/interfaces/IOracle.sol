// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IOracle {
    /**
     * @notice Create new oracle request
     * @param _requestId - ID of the Oracle Request
     * @param _jobId the job specification ID that the request is created for
     * @param _nonce represents the number of requested job
     * @param _callbackFunctionId function to use for callback
     * @param _data - Return data for fulfilment
     */
    function createRequest(
        bytes32 _requestId,
        bytes32 _jobId,
        uint256 _nonce,
        bytes4 _callbackFunctionId,
        bytes calldata _data
    ) external;

    /**
     * @notice Cancelling oracle request
     * @param _requestId - ID of the Oracle Request
     * @param _callbackFunctionId - Return functionID callback
     */
    function cancelRequest(bytes32 _requestId, bytes4 _callbackFunctionId) external;
}
