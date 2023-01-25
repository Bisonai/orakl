// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

abstract contract RequestResponseConsumerBase {
    error OnlyCoordinatorCanFulfill(address have, address want);
    address private immutable requestResponseCoordinator;

    /**
     * @param _requestResponseCoordinator address of RequestResponseCoordinator contract
     */
    constructor(address _requestResponseCoordinator) {
        requestResponseCoordinator = _requestResponseCoordinator;
    }

    function fulfillRequest(uint256 requestId, uint256 response) internal virtual;

    function rawFulfillRequest(uint256 requestId, uint256 response) external {
        if (msg.sender != requestResponseCoordinator) {
            revert OnlyCoordinatorCanFulfill(msg.sender, requestResponseCoordinator);
        }
        fulfillRequest(requestId, response);
    }
}
