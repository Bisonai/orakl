// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IOracle {
    /**
     * @notice Function to create a new oracle request
     */
    function createNewRequest(
        bytes32 _requestId,
        bytes32 _jobId,
        uint256 _nonce,
        address _callbackAddress,
        bytes4 _callbackFunctionId,
        bytes calldata _data
    ) external;

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure returns (string memory);

    /**
     * @notice Cancelling Oracle Request
     * @param _requestId - ID of the Oracle Request
     * @param _callbackAddress - Callback Address of Oracle Cancellation
     * @param _callbackFunctionId - Return functionID callback
     */
    function cancelOracleRequest(bytes32 _requestId, address _callbackAddress, bytes4 _callbackFunctionId) external;
}
