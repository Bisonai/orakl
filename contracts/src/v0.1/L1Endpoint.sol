// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./VRFConsumerBase.sol";
import "./interfaces/IVRFCoordinator.sol";
import "./interfaces/IRegistry.sol";

contract L1Endpoint is Ownable, VRFConsumerBase {
    IVRFCoordinator COORDINATOR;
    IRegistry public registry; // Reference to the Registry contract
    mapping(address => bool) private _oracles;
    mapping(uint256 => address) private _requestIdRequester;
    uint256 private _fee;

    error InsufficientBalance();
    error OnlyOracle();
    error FailedToDeposit();
    error ConsumerValid();

    event FeeUpdated(uint256 oldFee, uint256 newFee);
    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);

    event RandomWordRequested(uint256 requestId, address requester);
    event RandomWordFulfilled(uint256 requestId, address requester, uint256[] randomWords);

    constructor(address coordinator, address registryAddress) VRFConsumerBase(coordinator) {
        COORDINATOR = IVRFCoordinator(coordinator);
        registry = IRegistry(registryAddress);
    }

    receive() external payable {}

    function setFee(uint256 newFee) public onlyOwner {
        uint256 cFee = _fee;
        _fee = newFee;
        emit FeeUpdated(cFee, newFee);
    }

    function addOracle(address oracle) public onlyOwner {
        _oracles[oracle] = true;
        emit OracleAdded(oracle);
    }

    function removeOracle(address oracle) public onlyOwner {
        delete _oracles[oracle];
        emit OracleRemoved(oracle);
    }

    function requestRandomWordsDirectPayment(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        uint256 accId,
        address consumer
    ) public returns (uint256) {
        if (!_oracles[msg.sender]) {
            revert OnlyOracle();
        }
        //check consumer and balance
        bool isValidConsumer = registry.isValidConsumer(accId, consumer);
        if (!isValidConsumer) {
            revert ConsumerValid();
        }
        uint256 balance = registry.getBalance(accId);
        if (balance < _fee) {
            revert InsufficientBalance();
        }
        //decrease balance
        registry.decreaseBalance(accId, _fee);
        uint256 requestId = COORDINATOR.requestRandomWords{value: _fee}(
            keyHash,
            callbackGasLimit,
            numWords,
            address(this)
        );
        _requestIdRequester[requestId] = consumer;
        emit RandomWordRequested(requestId, consumer);
        return requestId;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) internal virtual override {
        emit RandomWordFulfilled(requestId, _requestIdRequester[requestId], randomWords);
        delete _requestIdRequester[requestId];
    }
}
