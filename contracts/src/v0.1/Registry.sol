// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";

contract Registry is Ownable {
    error NotEnoughFee();
    error InvalidChainID();
    error InsufficientBalance();
    error NotFeePayerOwner();
    error NotChainOwner();

    event ChainProposed(address sender, uint chainID);
    event ChainConfirmed(uint256 chainID);
    event ProposeFeeSet(uint256 fee);
    event AggregatorAdded(uint256 chainID, uint256 aggregatorID);
    event AggregatorRemoved(uint256 chainID, uint256 aggregatorID);
    event FeePayerAdded(uint256 chainID, address feePayer);
    event FeePayerRemoved(uint256 chainID, address feePayer);
    event feeConsumerAdded(uint256 chaiID, address feePayer, address feeConsumer);

    uint256 public proposeFee;
    struct AggregatorPair {
        uint256 aggregatorID;
        address l1Aggregator;
        address l2Aggregator;
    }

    mapping(uint256 => AggregatorPair[]) public aggregators; // chain ID to aggregator pairs
    mapping(uint256 => uint256) public aggregatorCount; // count aggregator IDs
    mapping(address => address) feePayerOwner;
    mapping(uint256 => mapping(address => bool)) public feePayer;
    mapping(uint256 => mapping(address => mapping(address => bool))) public consumer;

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
    modifier onlyConfirmedChainOwner(uint256 chainId) {
        if (chainRegistry[chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        if (chainRegistry[chainId].owner != msg.sender) {
            revert NotChainOwner();
        }
        _;
    }

    function addFeeConsumer(
        uint256 chainID,
        address _feePayer,
        address _feeConsumer
    ) external onlyConfirmedChain(chainID) {
        if (feePayerOwner[_feePayer] != msg.sender) {
            revert NotFeePayerOwner();
        }
        consumer[chainID][_feePayer][_feeConsumer] = true;
        emit feeConsumerAdded(chainID, _feePayer, _feeConsumer);
    }

    function removeFeeConsumer(
        uint256 chainID,
        address _feePayer,
        address _feeConsumer
    ) external onlyConfirmedChain(chainID) {
        if (feePayerOwner[_feePayer] != msg.sender) {
            revert NotFeePayerOwner();
        }
        delete consumer[chainID][_feePayer][_feeConsumer];
    }

    function addFeePayer(uint256 chainID, address _feePayer) external onlyConfirmedChain(chainID) {
        feePayer[chainID][_feePayer] = true;
        feePayerOwner[_feePayer] = msg.sender;
        emit FeePayerAdded(chainID, _feePayer);
    }

    function removeFeePayer(
        uint256 chainID,
        address _feePayer
    ) external onlyConfirmedChain(chainID) {
        if (feePayerOwner[_feePayer] != msg.sender) {
            revert NotFeePayerOwner();
        }
        feePayer[chainID][_feePayer] = false;
        emit FeePayerRemoved(chainID, _feePayer);
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

    function proposeChain(
        uint256 _chainID,
        string memory _jsonRpc,
        address _endpoint
    ) external payable {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }
        pendingProposal[_chainID].jsonRpc = _jsonRpc;
        pendingProposal[_chainID].endpoint = _endpoint;
        pendingProposal[_chainID].owner = msg.sender;
        emit ChainProposed(msg.sender, _chainID);
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
}
