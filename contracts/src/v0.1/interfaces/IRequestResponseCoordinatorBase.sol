// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/Orakl.sol";

interface IRequestResponseCoordinatorBase {
    // RequestCommitment holds information sent from off-chain oracle
    // describing details of request.
    struct RequestCommitment {
        uint64 blockNum;
        uint64 accId;
        uint8 numSubmission;
        uint32 callbackGasLimit;
        address sender;
        bool isDirectPayment;
        bytes32 jobId;
    }

    /**
     * @notice Creates a request to RequestResponse oracle using a
     * [regular] account.
     * @dev Generates and stores a request ID, increments the local
     * nonce, creates a request on the target oracle contract.
     * @dev Emits Requested event.
     * @param req The initialized Request
     * @param callbackGasLimit - How much gas you'd like to receive in
     * your fulfillRequest callback. Note that gasleft() inside
     * fulfillRequest may be slightly less than this amount because of
     * gas used calling the function (argument decoding etc.), so you
     * may need to request slightly more than you expect to have
     * inside fulfillRequest. The acceptable range is [0, maxGasLimit]
     * @param accId - The ID of the account. Must be funded with the
     * minimum account balance.
     * @param numSubmission number of requested submission to compute
     * the final aggregate value
     @return requestId - A unique * identifier of the request. Can be
     used to match a request to a * response in fulfillRequest.
     */
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint64 accId,
        uint8 numSubmission
    ) external returns (uint256);

    /**
     * @notice Creates a request to RequestResponse oracle using a
     * [temporary] account.
     * @dev Generates and stores a request ID, increments the local
     * nonce, creates a request on the target oracle contract.
     * @dev Emits Requested event.
     * @param req The initialized Request
     * @param callbackGasLimit - How much gas you'd like to receive in
     * your fulfillRequest callback. Note that gasleft() inside
     * fulfillRequest may be slightly less than this amount because of
     * gas used calling the function (argument decoding etc.), so you
     * may need to request slightly more than you expect to have
     * inside fulfillRequest. The acceptable range is [0, maxGasLimit]
     * @param numSubmission number of requested submission to compute
     * the final aggregate value
     * @param refundRecipient recipient of an extra $KLAY amount that
     * was sent together with service request
     * @return requestId - A unique identifier of the request. Can be
     * used to match a request to a response in fulfillRequest.
     */
    function requestData(
        Orakl.Request memory req,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        address refundRecipient
    ) external payable returns (uint256);

    function fulfillDataRequestUint128(
        uint256 requestId,
        uint128 response,
        RequestCommitment memory rc
    ) external;

    function fulfillDataRequestInt256(
        uint256 requestId,
        int256 response,
        RequestCommitment memory rc
    ) external;

    function fulfillDataRequestBool(
        uint256 requestId,
        bool response,
        RequestCommitment memory rc
    ) external;

    function fulfillDataRequestString(
        uint256 requestId,
        string memory response,
        RequestCommitment memory rc
    ) external;

    function fulfillDataRequestBytes32(
        uint256 requestId,
        bytes32 response,
        RequestCommitment memory rc
    ) external;

    function fulfillDataRequestBytes(
        uint256 requestId,
        bytes memory response,
        RequestCommitment memory rc
    ) external;

    /**
     * @notice Different jobs specified by jobId have allowed
     * different number of of requests for submissions that depends on
     * total number of registered oracles.
     */
    function validateNumSubmission(bytes32 jobId, uint8 numSubmission) external;
}
