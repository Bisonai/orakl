// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/Orakl.sol";

interface IRequestResponseCoordinator {
    // RequestCommitment holds information sent from off-chain oracle
    // describing details of request.
    struct RequestCommitment {
        uint64 blockNum;
        uint64 accId;
        uint32 callbackGasLimit;
        address sender;
    }

    /**
     * @notice Creates a request to RequestResponse oracle
     * @dev Generates and stores a request ID, increments the local nonce, creates a request on the target oracle contract.
     * @dev Emits Requested event.
     * @param req The initialized Request
     * @param callbackGasLimit - How much gas you'd like to receive in your
     * fulfillRequest callback. Note that gasleft() inside fulfillRequest
     * may be slightly less than this amount because of gas used calling the function
     * (argument decoding etc.), so you may need to request slightly more than you expect
     * to have inside fulfillRequest. The acceptable range is [0, maxGasLimit]
     * @param accId  - The ID of the account. Must be funded
     * with the minimum account balance required for the selected keyHash.
     * @return requestId - A unique identifier of the request. Can be used to match
     * a request to a response in fulfillRequest.
     */
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numbSubmission
    ) external returns (uint256);

    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint8 numbSubmission
    ) external payable returns (uint256);

    /**
     * @notice Canceling oracle request
     * @param requestId - ID of the Oracle Request
     */
    function cancelRequest(uint256 requestId) external;

    function fulfillDataRequestUint256(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);
}
