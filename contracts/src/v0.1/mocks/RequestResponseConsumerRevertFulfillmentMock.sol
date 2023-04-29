// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../RequestResponseConsumerFulfill.sol";
import "../RequestResponseConsumerBase.sol";

/**
 * @notice RequestResponseConsumerRevertFulfillmentMock contract is
 * used only for testing whether oracle receives payment when
 * fulfillment function fails. This contract is missing safety
 * features that should otherwise be applied.
 */
contract RequestResponseConsumerRevertFulfillmentMock is RequestResponseConsumerFulfillUint128 {
    using Orakl for Orakl.Request;
    error AnyError();

    constructor(address coordinator) RequestResponseConsumerBase(coordinator) {}

    function requestDataUint128(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission
    ) public returns (uint256) {
        bytes32 jobId = keccak256(abi.encodePacked("uint128"));
        Orakl.Request memory req = buildRequest(jobId);
        return COORDINATOR.requestData(req, callbackGasLimit, accId, numSubmission);
    }

    function fulfillDataRequest(
        uint256 /*requestId*/,
        uint128 /*response*/
    ) internal pure override {
        revert AnyError();
    }
}
