// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./VRFConsumerBase.sol";
import "./interfaces/IVRFCoordinator.sol";

contract L1Endpoint is Ownable, VRFConsumerBase {
    IVRFCoordinator COORDINATOR;
    mapping(address => uint256) private _balance;
    mapping(address => bool) private _oracles;
    mapping(uint256 => address) private _requestIdRequester;
    uint256 private _fee;

    error InsufficientBalance();
    error OnlyOracle();

    event FeeUpdated(uint256 oldFee, uint256 newFee);
    event OracleAdded(address oracle);
    event OracleRemoved(address oracle);

    event RandomWordRequested(uint256 requestId, address requester);
    event RandomWordFulfilled(uint256 requestId, address requester, uint256[] randomWords);

    constructor(address coordinator) VRFConsumerBase(coordinator) {
        COORDINATOR = IVRFCoordinator(coordinator);
    }

    receive() external payable {
        _balance[msg.sender] += msg.value;
    }

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
        address payer,
        address requester
    ) public returns (uint256 requestId) {
        if (!_oracles[msg.sender]) {
            revert OnlyOracle();
        }
        if (_balance[payer] < _fee) {
            revert InsufficientBalance();
        }
        _balance[payer] = _balance[payer] - _fee;

        uint256 id = COORDINATOR.requestRandomWords{value: _fee}(
            keyHash,
            callbackGasLimit,
            numWords,
            msg.sender
        );
        _requestIdRequester[id] = requester;
        emit RandomWordRequested(id, requester);
        return id;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) internal virtual override {
        emit RandomWordFulfilled(requestId, _requestIdRequester[requestId], randomWords);
        delete _requestIdRequester[requestId];
    }
}
