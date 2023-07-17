// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";

contract Registry is Ownable {
    error NotEnoughFee();
    error InvalidChainID();
    error InsufficientBalance();

    event ChainProposed(address sender, uint chainID);
    event ChainConfirmed(uint256 chainID);
    event ProposeFeeSet(uint256 fee);

    uint256 public proposeFee;
    struct AggregatorPair {
        address l1Aggregator;
        address l2Aggregator;
    }

    struct L2Endpoint {
        string jsonRpc;
        address endpoint;
        address owner;
        uint256 startRound; // for datafeeds
        AggregatorPair[] aggregatorPair;
    }
    // chainId => L2 Endpoint
    mapping(uint256 => L2Endpoint) public chainRegistry;
    // pending proposal
    mapping(uint256 => L2Endpoint) pendingProposal;

    // L2 consumer => L1 payer
    // Can be updated only by Orakl Network through call from L2
    // mapping(address => address) accountRegistry;

    function proposeChain(
        uint256 _chainID,
        string memory _jsonRpc,
        address _endpoint,
        uint256 _startRound,
        address _l1Aggregator,
        address _l2Aggregator
    ) external payable {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }
        pendingProposal[_chainID].jsonRpc = _jsonRpc;
        pendingProposal[_chainID].endpoint = _endpoint;
        pendingProposal[_chainID].owner = msg.sender;
        pendingProposal[_chainID].startRound = _startRound;
        AggregatorPair memory pair = AggregatorPair({
            l1Aggregator: _l1Aggregator,
            l2Aggregator: _l2Aggregator
        });
        pendingProposal[_chainID].aggregatorPair.push(pair);
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
