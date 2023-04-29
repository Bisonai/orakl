// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../VRFConsumerBase.sol";
import "../interfaces/IVRFCoordinator.sol";

/**
 * @notice VRFConsumerRevertFulfillmentMock contract is used only for
 * testing whether oracle receives payment when fulfillment function
 * fails. This contract is missing safety features that should
 * otherwise be applied.
 */
contract VRFConsumerRevertFulfillmentMock is VRFConsumerBase {
    IVRFCoordinator COORDINATOR;
    error AnyError();

    constructor(address coordinator) VRFConsumerBase(coordinator) {
        COORDINATOR = IVRFCoordinator(coordinator);
    }

    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) public returns (uint256) {
        return COORDINATOR.requestRandomWords(keyHash, accId, callbackGasLimit, numWords);
    }

    function fulfillRandomWords(
        uint256 /* requestId */,
        uint256[] memory /*randomWords*/
    ) internal pure override {
        revert AnyError();
    }
}
