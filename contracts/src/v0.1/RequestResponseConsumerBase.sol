// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/RequestResponseCoordinatorInterface.sol";

abstract contract RequestResponseConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);
    address private immutable requestResponseCoordinator;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        requestResponseCoordinator = _requestResponseCoordinator;
    }

    /**
     * @notice Build a request using the Orakl library
     * @param jobId the job specification ID that the request is created for
     * @return req request in memory
     */
    function buildRequest(bytes32 jobId) internal view returns (Orakl.Request memory req) {
        return
            req.initialize(
                jobId,
                requestResponseCoordinator,
                RequestResponseCoordinatorInterface(requestResponseCoordinator)
                    .fulfillRequest
                    .selector
            );
    }

    /**
     * @notice Pass predefined request to RequestResponseCoordinator.
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
    ) internal returns (uint256 requestId) {
        requestId = RequestResponseCoordinatorInterface(requestResponseCoordinator).sendRequest(
            req,
            accId,
            requestConfirmations,
            callbackGasLimit
        );
    }

    function fulfillRequest(uint256 requestId, uint256 response) internal virtual;

    function rawFulfillRequest(uint256 requestId, uint256 response) external {
        if (msg.sender != requestResponseCoordinator) {
            revert OnlyCoordinatorCanFulfill(msg.sender, requestResponseCoordinator);
        }
        fulfillRequest(requestId, response);
    }
}
