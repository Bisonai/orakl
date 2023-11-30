// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./L1EndpointBase.sol";
import "./interfaces/IVRFCoordinator.sol";

abstract contract L1EndpointVRF is L1EndpointBase {
    IVRFCoordinator VRFCOORDINATOR;

    error OnlyVRFCoordinatorCanFulfill(address have, address want);
    event RandomWordRequested(uint256 requestId, address sender);
    event RandomWordFulfilled(
        uint256 requestId,
        uint256 l2RequestId,
        address sender,
        uint256 callbackGasLimit,
        uint256[] randomWords
    );

    constructor(address _vrfCoordinator) {
        VRFCOORDINATOR = IVRFCoordinator(_vrfCoordinator);
    }

    function rawFulfillRandomWords(uint256 requestId, uint256[] memory randomWords) external {
        address coordinatorAddress = address(VRFCOORDINATOR);
        if (msg.sender != address(coordinatorAddress)) {
            revert OnlyVRFCoordinatorCanFulfill(msg.sender, coordinatorAddress);
        }
        fulfillRandomWords(requestId, randomWords);
    }

    function requestRandomWords(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        uint64 accId,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256) {
        uint64 reqCount = 0;
        uint8 numSubmission = 1;
        uint256 fee = VRFCOORDINATOR.estimateFee(reqCount, numSubmission, callbackGasLimit);
        pay(accId, sender, fee);
        uint256 requestId = VRFCOORDINATOR.requestRandomWords{value: fee}(
            keyHash,
            callbackGasLimit,
            numWords,
            address(this)
        );
        sRequest[requestId] = RequestDetail(l2RequestId, sender, callbackGasLimit);
        emit RandomWordRequested(requestId, sender);
        return requestId;
    }

    function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) internal {
        emit RandomWordFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
            randomWords
        );
        delete sRequest[requestId];
    }
}
