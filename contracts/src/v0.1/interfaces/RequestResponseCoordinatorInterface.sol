// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../libraries/Orakl.sol";

interface RequestResponseCoordinatorInterface {
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
     * @param accId  - The ID of the account. Must be funded
     * with the minimum account balance required for the selected keyHash.
     * @param requestConfirmations - How many blocks you'd like the
     * oracle to wait before responding to the request. See SECURITY CONSIDERATIONS
     * for why you may want to request more. The acceptable range is
     * [minimumRequestBlockConfirmations, 200].
     * @param callbackGasLimit - How much gas you'd like to receive in your
     * fulfillRequest callback. Note that gasleft() inside fulfillRequest
     * may be slightly less than this amount because of gas used calling the function
     * (argument decoding etc.), so you may need to request slightly more than you expect
     * to have inside fulfillRequest. The acceptable range is
     * [0, maxGasLimit]
     * @return requestId - A unique identifier of the request. Can be used to match
     * a request to a response in fulfillRequest.
   */
    function sendRequest(
        Orakl.Request memory req,
        uint64 accId,
        uint16 requestConfirmations,
        uint32 callbackGasLimit
    ) external returns (uint256);

    function sendRequestPayment(
        Orakl.Request memory req,
        uint16 requestConfirmations,
        uint32 callbackGasLimit
    ) external payable returns (uint256);

    /**
     * @notice Cancelling oracle request
     * @param requestId - ID of the Oracle Request
     */
    function cancelRequest(uint256 requestId) external;

    function fulfillRequest(
        uint256 requestId,
        uint256 response,
        RequestCommitment memory rc,
        bool isDirectPayment
    ) external returns (uint256);
}
