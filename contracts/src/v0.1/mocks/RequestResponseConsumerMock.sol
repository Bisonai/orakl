// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

/* import 'hardhat/console.sol'; */
import "../RequestResponseConsumerBase.sol";
import '../interfaces/RequestResponseCoordinatorInterface.sol';

contract RequestResponseConsumerMock is RequestResponseConsumerBase {
    using Orakl for Orakl.Request;
    uint256 public s_response;
    address private s_owner;

    RequestResponseCoordinatorInterface immutable COORDINATOR;

    error OnlyOwner(address notOwner);

    modifier onlyOwner() {
        if (msg.sender != s_owner) {
            revert OnlyOwner(msg.sender);
        }
        _;
    }

    constructor(address coordinator) RequestResponseConsumerBase(coordinator) {
        s_owner = msg.sender;
        COORDINATOR = RequestResponseCoordinatorInterface(coordinator);
    }

    // Receive remaining payment from requestDataPayment
    receive() external payable {}

    function requestData(
      uint64 accId,
      uint16 requestConfirmations,
      uint32 callbackGasLimit
    )
        public
        onlyOwner
        returns (uint256 requestId)
    {
        bytes32 jobId = keccak256(abi.encodePacked("any-api-int256"));
        /* console.log('in2'); */
        // FIXME!!
        Orakl.Request memory req = COORDINATOR.buildRequest(jobId);
        /* console.log('in3'); */
        /* req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD"); */
        /* req.add("path", "RAW,ETH,USD,PRICE"); */

        /* console.log('requestData req.buf.buf.length %s', req.buf.buf.length); */

        /* bytes memory hello = req.buf.buf; */
        /* bytes memory tmp; */
        /* assembly { */
        /*   tmp := hello */
        /* } */
        /* console.log('requestData %s', string(tmp)); */
        /* console.log('problem solved'); */

        req.add("g", "g");
        /* console.log('in3'); */

        /* requestId = COORDINATOR.sendRequest( */
        /*     req, */
        /*     accId, */
        /*     requestConfirmations, */
        /*     callbackGasLimit */
        /* ); */
    }

    function cancelRequest(uint256 requestId) public onlyOwner {
        COORDINATOR.cancelRequest(requestId);
    }

    function fulfillRequest(
        uint256 /*requestId*/,
        uint256 response
    )
        internal
        override
    {
        s_response = response;
    }
}
