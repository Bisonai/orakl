// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "./RequestResponseConsumerBase.sol";
import "./L1EndpointBase.sol";
import "./RequestResponseConsumerFulfill.sol";

abstract contract L1EndpointRequestResponse is
    RequestResponseConsumerBase,
    L1EndpointBase,
    RequestResponseConsumerFulfillUint128
{
    using Orakl for Orakl.Request;
    event DataRequested(uint256 requestId, address sender);
    event DataRequestFulfilled(
        uint256 requestId,
        uint256 l2RequestId,
        address sender,
        uint256 callbackGasLimit,
        uint128 response
    );

    constructor(
        address requestResponseCoordinator
    ) RequestResponseConsumerBase(requestResponseCoordinator) {}

    //request data
    function requestData(
        uint64 accId,
        uint32 callbackGasLimit,
        uint8 numSubmission,
        bytes32 jobId,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256 requestId) {
        if (!sOracles[msg.sender]) {
            revert OnlyOracle();
        }
        //check consumer and balance
        bool isValidConsumer = REGISTRY.isValidConsumer(accId, sender);
        if (!isValidConsumer) {
            revert ConsumerValid();
        }
        uint256 balance = REGISTRY.getBalance(accId);
        uint64 reqCount = 0;
        uint256 fee = COORDINATOR.estimateFee(reqCount, 1, callbackGasLimit);
        REGISTRY.decreaseBalance(accId, fee);
        if (balance < fee) {
            revert InsufficientBalance();
        }

        Orakl.Request memory req = buildRequest(jobId);
        //change here for your expected data
        req.add(
            "get",
            "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD"
        );
        req.add("path", "RAW,KLAY,USD,PRICE");
        req.add("pow10", "8");

        uint256 id = COORDINATOR.requestData{value: fee}(
            req,
            callbackGasLimit,
            numSubmission,
            address(this)
        );
        sRequest[id] = RequestDetail(l2RequestId, sender, callbackGasLimit);
        emit DataRequested(requestId, sender);
        return requestId;
    }

    function fulfillDataRequest(uint256 requestId, uint128 response) internal override {
        emit DataRequestFulfilled(
            requestId,
            sRequest[requestId].l2RequestId,
            sRequest[requestId].sender,
            sRequest[requestId].callbackGasLimit,
            response
        );
        delete sRequest[requestId];
    }
}
