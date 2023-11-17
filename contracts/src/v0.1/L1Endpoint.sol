// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./VRFConsumerBase.sol";
import "./interfaces/IVRFCoordinator.sol";
import "./interfaces/IRegistry.sol";

contract L1Endpoint is Ownable, VRFConsumerBase {
    IVRFCoordinator COORDINATOR;
    IRegistry public REGISTRY; // Reference to the Registry contract

    struct RequestDetail {
        uint256 l2RequestId;
        address sender;
        uint256 callbackGasLimit;
    }
    mapping(address => bool) private sOracles;
    mapping(uint256 => RequestDetail) private sRequest;

    error InsufficientBalance();
    error OnlyOracle();
    error FailedToDeposit();
    error ConsumerValid();

    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);

    event RandomWordRequested(uint256 requestId, address sender);
    event RandomWordFulfilled(
        uint256 requestId,
        uint256 l2RequestId,
        address sender,
        uint256 callbackGasLimit,
        uint256[] randomWords
    );

    constructor(address coordinator, address registryAddress) VRFConsumerBase(coordinator) {
        COORDINATOR = IVRFCoordinator(coordinator);
        REGISTRY = IRegistry(registryAddress);
    }

    receive() external payable {}

    function addOracle(address oracle) public onlyOwner {
        sOracles[oracle] = true;
        emit OracleAdded(oracle);
    }

    function removeOracle(address oracle) public onlyOwner {
        delete sOracles[oracle];
        emit OracleRemoved(oracle);
    }

    function requestRandomWords(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        uint256 accId,
        address sender,
        uint256 l2RequestId
    ) public returns (uint256) {
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
        uint8 numSubmission = 1;
        uint256 fee = COORDINATOR.estimateFee(reqCount, numSubmission, callbackGasLimit);
        if (balance < fee) {
            revert InsufficientBalance();
        }

        //decrease balance
        REGISTRY.decreaseBalance(accId, fee);
        uint256 requestId = COORDINATOR.requestRandomWords{value: fee}(
            keyHash,
            callbackGasLimit,
            numWords,
            address(this)
        );
        sRequest[requestId] = RequestDetail(l2RequestId, sender, callbackGasLimit);
        emit RandomWordRequested(requestId, sender);
        return requestId;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) internal virtual override {
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
