// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./VRFConsumerBase.sol";
import "./interfaces/IVRFCoordinator.sol";
import "./interfaces/IRegistry.sol";

contract L1Endpoint is Ownable, VRFConsumerBase, IRegistry {
    IVRFCoordinator COORDINATOR;
    IRegistry public registry; // Reference to the Registry contract
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

    constructor(address coordinator, address _registry) VRFConsumerBase(coordinator) {
        COORDINATOR = IVRFCoordinator(coordinator);
        registry = IRegistry(_registry);
    }

    receive() external payable {
        _balance[msg.sender] += msg.value;
    }
    function deposit(uint256 _accId, uint256 _amount) external payable  {
        require(isValidOwnerAndConsumer(_accId, msg.sender, msg.sender), "Not a valid fee payer or consumer");
        require(msg.value == _amount, "Invalid deposit amount");
        payable(address(this)).transfer(_amount);
        registry.increaseBalance(_accId, _amount);
    }
    function increaseBalance(uint256 _accId, uint256 _amount) external override {
        registry.increaseBalance(_accId, _amount);
    }

    function decreaseBalance(uint256 _accId, uint256 _amount) external override {
        registry.decreaseBalance(_accId, _amount);
    }

    function isValidConsumer(
        uint256 _accId,
        address _consumer
    ) public view override returns (bool) {
        IRegistry.Account memory account = registry.getAccount(_accId);
        for (uint8 i = 0; i < account.consumerCount; i++) {
            if (account.consumers[i] == _consumer) {
                return true;
            }
        }
        return false;
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

    function isValidOwnerAndConsumer(
        uint256 _accId,
        address _owner,
        address _consumer
    ) public view returns (bool) {
        IRegistry.Account memory account = registry.getAccount(_accId);
        return (account.owner == _owner) && registry.isValidConsumer(_accId, _consumer);
    }

    function requestRandomWordsDirectPayment(
        bytes32 keyHash,
        uint32 callbackGasLimit,
        uint32 numWords,
        uint256 accId,
        address payer,
        address consumer
    ) public returns (uint256 requestId) {
        require(isValidOwnerAndConsumer(accId, payer, consumer), "Invalid Owner or consumer");
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
        _requestIdRequester[id] = consumer;
        emit RandomWordRequested(id, consumer);
        return id;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) internal virtual override {
        emit RandomWordFulfilled(requestId, _requestIdRequester[requestId], randomWords);
        delete _requestIdRequester[requestId];
    }

    function getAccount(uint256 _accId) external view override returns (Account memory) {}

}
