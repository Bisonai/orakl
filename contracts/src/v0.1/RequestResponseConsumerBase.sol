// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/RequestResponseCoordinatorInterface.sol";

abstract contract RequestResponseConsumerBase {
    using Orakl for Orakl.Request;

    error OnlyCoordinatorCanFulfill(address have, address want);
    RequestResponseCoordinatorInterface public immutable COORDINATOR;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        COORDINATOR = RequestResponseCoordinatorInterface(_requestResponseCoordinator);
    }

    /**
     * @notice Build a request using the Orakl library
     * @param jobId the job specification ID that the request is created for
     * @return req request in memory
     */
    function buildRequest(bytes32 jobId) internal view returns (Orakl.Request memory req) {
        return req.initialize(jobId, address(COORDINATOR), COORDINATOR.fulfillDataRequest.selector);
    }

    function fulfillDataRequest(uint256 requestId, uint256 response) internal virtual;

    function rawFulfillDataRequest(uint256 requestId, uint256 response) external {
        address coordinatorAddress = address(COORDINATOR);
        if (msg.sender != coordinatorAddress) {
            revert OnlyCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }
        fulfillDataRequest(requestId, response);
    }
}
