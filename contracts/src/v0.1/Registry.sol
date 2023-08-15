// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";

contract Registry is Ownable {
    error NotEnoughFee();
    error InvalidChainID();
    error InsufficientBalance();
    error NotFeePayerOwner();
    error NotChainOwner();
    error ChainExisted();

    event ChainProposed(address sender, uint chainID);
    event ChainConfirmed(uint256 chainID);
    event ProposeFeeSet(uint256 fee);
    event AggregatorAdded(uint256 chainID, uint256 aggregatorID);
    event AggregatorRemoved(uint256 chainID, uint256 aggregatorID);
    event AccountCreated(uint256 accId, uint256 chainId, address owner);
    event AccountUpdated(uint256 accId, uint256 chainId, address owner);
    event AccountDeleted(uint256 accId);
    event ConsumerAdded(uint256 accId, address consumerAddress);
    event ConsumerRemoved(uint256 accId, address consumerAddress);
    event BalanceIncreased(uint256 _accId, uint256 _amount);
    event BalanceDecreased(uint256 _accId, uint256 _amount);

    uint256 public proposeFee;
    struct AggregatorPair {
        uint256 aggregatorID;
        address l1Aggregator;
        address l2Aggregator;
    }

    mapping(uint256 => AggregatorPair[]) public aggregators; // chain ID to aggregator pairs
    mapping(uint256 => uint256) public aggregatorCount; // count aggregator IDs
    // mapping(uint256 => mapping(address => address)) public feePayer; // chainID -> owner -> feepayer

    // mapping(uint256 => mapping(address => mapping(address => bool))) public consumer; // chainID -> owner -> payer -> consumer
    // mapping(uint => address) accountConsmers;

    mapping(uint256 => Account) private accounts;
    address public l1Endpoint;
    uint256 private nextAccountId = 1;
    struct Account {
        uint256 accId;
        uint256 chainId;
        address owner;
        address feePayer;
        address[100] consumers;
        uint8 consumerCount;
        uint256 balance;
    }

    struct L2Endpoint {
        uint256 _chainID;
        string jsonRpc;
        address endpoint;
        address owner;
    }
    // chainId => L2 Endpoint
    mapping(uint256 => L2Endpoint) public chainRegistry;
    // pending proposal
    mapping(uint256 => L2Endpoint) pendingProposal;

    modifier onlyConfirmedChain(uint256 chainId) {
        if (chainRegistry[chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        _;
    }
    modifier onlyAccountOwner(uint256 _accId) {
        require(_accId > 0 && _accId < nextAccountId, "Account does not exist");
        require(accounts[_accId].owner == msg.sender, "Not the account owner");
        _;
    }
    modifier onlyConfirmedChainOwner(uint256 chainId) {
        if (chainRegistry[chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        if (chainRegistry[chainId].owner != msg.sender) {
            revert NotChainOwner();
        }
        _;
    }

    modifier onlyL1Endpoint() {
        require(msg.sender == l1Endpoint, "Only L1Endpoint contract can call this");
        _;
    }

    constructor(address _l1Endpoint) {
        l1Endpoint = _l1Endpoint;
    }


    function createAccount(uint256 _chainId, address _owner, address _feePayer) external {
        if (chainRegistry[_chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        Account storage newAccount = accounts[nextAccountId];
        newAccount.accId = nextAccountId;
        newAccount.chainId = _chainId;
        newAccount.owner = _owner;
        newAccount.feePayer = _feePayer;
        newAccount.balance = 0; 

        emit AccountCreated(nextAccountId, _chainId, _owner);
        nextAccountId++;
    }

    function increaseBalance(uint256 _accId, uint256 _amount) external onlyL1Endpoint {
        accounts[_accId].balance += _amount;
        emit BalanceIncreased(_accId, _amount);
    }
    function decreaseBalance(uint256 _accId, uint256 _amount) external onlyL1Endpoint {
        require(accounts[_accId].balance >= _amount, "Insufficient balance");
        accounts[_accId].balance -= _amount;
        emit BalanceDecreased(_accId, _amount);
    }

    function deleteAccount(uint256 _accId) external onlyAccountOwner(_accId) {
        delete accounts[_accId];
        emit AccountDeleted(_accId);
    }

    function addConsumer(
        uint256 _accId,
        address _consumerAddress
    ) external onlyAccountOwner(_accId) {
        Account storage account = accounts[_accId];
        require(account.consumerCount < 100, "Max consumers reached");

        account.consumers[account.consumerCount] = _consumerAddress;
        account.consumerCount++;

        emit ConsumerAdded(_accId, _consumerAddress);
    }

    function removeConsumer(
        uint256 _accId,
        address _consumerAddress
    ) external onlyAccountOwner(_accId) {
        require(_accId > 0 && _accId < nextAccountId, "Account does not exist");
        Account storage account = accounts[_accId];

        for (uint8 i = 0; i < account.consumerCount; i++) {
            if (account.consumers[i] == _consumerAddress) {
                account.consumerCount--;
                account.consumers[i] = account.consumers[account.consumerCount];
                account.consumers[account.consumerCount] = address(0);

                emit ConsumerRemoved(_accId, _consumerAddress);
                return;
            }
        }
    }

    function addAggregator(
        uint256 chainID,
        address l1Aggregator,
        address l2Aggregator
    ) external onlyConfirmedChainOwner(chainID) {
        AggregatorPair memory newAggregatorPair = AggregatorPair({
            aggregatorID: aggregatorCount[chainID]++,
            l1Aggregator: l1Aggregator,
            l2Aggregator: l2Aggregator
        });
        aggregators[chainID].push(newAggregatorPair);
        emit AggregatorAdded(chainID, newAggregatorPair.aggregatorID);
    }

    function removeAggregator(
        uint256 chainID,
        uint256 aggregatorID
    ) external onlyConfirmedChainOwner(chainID) {
        AggregatorPair[] storage aggregatorInfo = aggregators[aggregatorID];
        for (uint256 i = 0; i < aggregatorInfo.length; i++) {
            if (aggregatorInfo[i].aggregatorID == aggregatorID) {
                // Move the last item to the current index to be removed
                aggregatorInfo[i] = aggregatorInfo[aggregatorInfo.length - 1];

                // Remove the last item from the list
                aggregatorInfo.pop();

                emit AggregatorRemoved(chainID, aggregatorID);
                break; // Exit the loop once item is found and removed
            }
        }
    }

    function proposeNewChain(
        uint256 _chainID,
        string memory _jsonRpc,
        address _endpoint
    ) external payable {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }
        if (chainRegistry[_chainID].owner != address(0)) {
            revert ChainExisted();
        }
        pendingProposal[_chainID].jsonRpc = _jsonRpc;
        pendingProposal[_chainID].endpoint = _endpoint;
        pendingProposal[_chainID].owner = msg.sender;
        emit ChainProposed(msg.sender, _chainID);
    }

    function editChainInfo(
        uint256 _chainID,
        string memory _jsonRpc,
        address _endpoint
    ) external payable {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }
        if (chainRegistry[_chainID].owner != msg.sender) {
            revert NotChainOwner();
        }
        chainRegistry[_chainID].jsonRpc = _jsonRpc;
        chainRegistry[_chainID].endpoint = _endpoint;
        chainRegistry[_chainID].owner = msg.sender;
        emit ChainConfirmed(_chainID);
    }

    function setProposeFee(uint256 _fee) public onlyOwner {
        proposeFee = _fee;
        emit ProposeFeeSet(_fee);
    }

    function confirmChain(uint256 _chainId) public onlyOwner {
        if (pendingProposal[_chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        chainRegistry[_chainId] = pendingProposal[_chainId];
        delete pendingProposal[_chainId];
        emit ChainConfirmed(_chainId);
    }

    receive() external payable {}

    function withdraw(uint256 _amount) external onlyOwner returns (bool) {
        uint256 balance = address(this).balance;
        if (balance < _amount) {
            revert InsufficientBalance();
        }
        (bool sent, ) = payable(msg.sender).call{value: _amount}("");
        return sent;
    }

    function getAccount(uint256 _accId) external view returns (Account memory) {
        require(_accId > 0 && _accId < nextAccountId, "Account does not exist");
        return accounts[_accId];
    }

    function getAccountsByChain(uint256 _chainId) external view returns (Account[] memory) {
        uint256 count = 0;
        for (uint256 i = 1; i < nextAccountId; i++) {
            if (accounts[i].chainId == _chainId) {
                count++;
            }
        }

        Account[] memory result = new Account[](count);
        uint256 index = 0;
        for (uint256 i = 1; i < nextAccountId; i++) {
            if (accounts[i].chainId == _chainId) {
                result[index] = accounts[i];
                index++;
            }
        }

        return result;
    }

    function getAccountsByOwner(address _owner) external view returns (Account[] memory) {
        uint256 count = 0;
        for (uint256 i = 1; i < nextAccountId; i++) {
            if (accounts[i].owner == _owner) {
                count++;
            }
        }

        Account[] memory result = new Account[](count);
        uint256 index = 0;
        for (uint256 i = 1; i < nextAccountId; i++) {
            if (accounts[i].owner == _owner) {
                result[index] = accounts[i];
                index++;
            }
        }

        return result;
    }
}
