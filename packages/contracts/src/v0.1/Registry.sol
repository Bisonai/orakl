// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IRegistry.sol";

contract Registry is Ownable, IRegistry {
    uint256 public constant MAX_CONSUMER = 100;
    error NotEnoughFee();
    error InvalidChainID();
    error InvalidAccId();
    error AccountExisted();

    error InsufficientBalance();
    error NotFeePayerOwner();
    error NotChainOwner();
    error ChainExisted();
    error DecreaseBalanceFailed();

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
    event BalanceIncreased(uint256 accId, uint256 amount);
    event BalanceDecreased(uint256 accId, uint256 amount);
    event L1EndpointSet(address oldEndpoint, address newEndpoint);
    event ChainEdited(string rpc, address endpoint);

    struct AggregatorPair {
        uint256 aggregatorID;
        address l1Aggregator;
        address l2Aggregator;
    }
    struct L2Endpoint {
        uint256 _chainID;
        string jsonRpc;
        address endpoint;
        address owner;
    }

    address public l1Endpoint;
    uint256 public proposeFee;
    uint256 private nextAccountId = 1;

    mapping(uint256 => AggregatorPair[]) public aggregators; // chain ID to aggregator pairs
    mapping(uint256 => uint256) public aggregatorCount; // count aggregator IDs
    mapping(uint256 => Account) private accounts;
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

    constructor() {}

    receive() external payable {}

    function setL1Endpoint(address _newAddress) public onlyOwner {
        address old = l1Endpoint;
        l1Endpoint = _newAddress;
        emit L1EndpointSet(old, l1Endpoint);
    }

    function setProposeFee(uint256 _fee) public onlyOwner {
        proposeFee = _fee;
        emit ProposeFeeSet(_fee);
    }

    function createAccount(uint256 _chainId) external onlyConfirmedChain(_chainId) {
        Account storage newAccount = accounts[nextAccountId];
        newAccount.accId = nextAccountId;
        newAccount.chainId = _chainId;
        newAccount.owner = msg.sender;
        newAccount.balance = 0;
        emit AccountCreated(nextAccountId, _chainId, msg.sender);
        nextAccountId++;
    }

    function deposit(uint256 _accId) public payable {
        accounts[_accId].balance += msg.value;
        emit BalanceIncreased(_accId, msg.value);
    }

    function decreaseBalance(uint256 _accId, uint256 _amount) external onlyL1Endpoint {
        if (accounts[_accId].balance < _amount) {
            revert InsufficientBalance();
        }
        accounts[_accId].balance -= _amount;
        (bool sent, ) = payable(msg.sender).call{value: _amount}("");
        if (!sent) {
            revert DecreaseBalanceFailed();
        }
        emit BalanceDecreased(_accId, _amount);
    }

    function getBalance(uint256 _accId) external view returns (uint256 balance) {
        balance = accounts[_accId].balance;
    }

    function getConsumer(uint256 _accId) external view returns (address[] memory consumers) {
        consumers = accounts[_accId].consumers;
    }

    function accountInfo(uint256 _accId) external view returns (uint256 balance, address owner) {
        return (accounts[_accId].balance, accounts[_accId].owner);
    }

    // thinking about needed or not
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

    function getLatestAccountIdByChain(uint256 _chainId) internal view returns (uint256) {
        uint256 latestAccId = 0;
        for (uint256 accId = 1; accId < nextAccountId; accId++) {
            if (accounts[accId].chainId == _chainId && accounts[accId].accId > latestAccId) {
                latestAccId = accounts[accId].accId;
            }
        }
        return latestAccId;
    }

    //

    function addConsumer(
        uint256 _accId,
        address _consumerAddress
    ) external onlyAccountOwner(_accId) {
        Account storage account = accounts[_accId];
        require(account.consumerCount < MAX_CONSUMER, "Max consumers reached");

        account.consumers.push(_consumerAddress);
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
            address[] storage consumers = account.consumers;
            if (consumers[i] == _consumerAddress) {
                account.consumerCount--;
                consumers[i] = consumers[account.consumerCount];
                consumers.pop();
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
        AggregatorPair[] storage aggregatorInfo = aggregators[chainID];
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
    ) external payable onlyConfirmedChainOwner(_chainID) {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }

        chainRegistry[_chainID].jsonRpc = _jsonRpc;
        chainRegistry[_chainID].endpoint = _endpoint;
        emit ChainEdited(_jsonRpc, _endpoint);
    }

    function confirmChain(uint256 _chainId) public onlyOwner {
        if (pendingProposal[_chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        chainRegistry[_chainId] = pendingProposal[_chainId];
        delete pendingProposal[_chainId];
        emit ChainConfirmed(_chainId);
    }

    function isValidConsumer(uint256 _accId, address _consumer) public view returns (bool) {
        Account memory account = accounts[_accId];
        for (uint8 i = 0; i < account.consumerCount; i++) {
            if (account.consumers[i] == _consumer) {
                return true;
            }
        }
        return false;
    }

    function withdraw(uint256 _amount) external onlyOwner returns (bool) {
        uint256 balance = address(this).balance;
        if (balance < _amount) {
            revert InsufficientBalance();
        }
        (bool sent, ) = payable(msg.sender).call{value: _amount}("");
        return sent;
    }
}
