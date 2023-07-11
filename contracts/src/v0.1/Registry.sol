// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";

contract Registry is Ownable {
    error NotEnoughFee();
    error InvalidChainID();

    event ChainProposed(address sender, uint chainID);
    event ChainConfirmed(uint chainID);
    event ProposeFeeSet(uint fee);

    uint256 public proposeFee;

    struct L2Endpoint {
        string jsonRpc;
        address endpoint;
        address owner;
        uint256 startRound; // for datafeeds
    }
    // chainId => L2 Endpoint
    mapping(uint256 => L2Endpoint) public chainRegistry;
    // pending proposal
    mapping(uint256 => L2Endpoint) pendingProposal;

    // L2 consumer => L1 payer
    // Can be updated only by Orakl Network through call from L2
    // mapping(address => address) accountRegistry;

    function proposeChain(
        uint _chainID,
        string memory _jsonRpc,
        address _endpoint,
        uint256 _startRound
    ) external payable {
        if (msg.value < proposeFee) {
            revert NotEnoughFee();
        }
        pendingProposal[_chainID].jsonRpc = _jsonRpc;
        pendingProposal[_chainID].endpoint = _endpoint;
        pendingProposal[_chainID].owner = msg.sender;
        pendingProposal[_chainID].startRound = _startRound;

        emit ChainProposed(msg.sender, _chainID);
    }

    function setProposeFee(uint _fee) public onlyOwner {
        proposeFee = _fee;
        emit ProposeFeeSet(_fee);
    }

    function confirmChain(uint _chainId) public onlyOwner {
        if (pendingProposal[_chainId].owner == address(0)) {
            revert InvalidChainID();
        }
        chainRegistry[_chainId] = pendingProposal[_chainId];
        delete pendingProposal[_chainId];
        emit ChainConfirmed(_chainId);
    }

    receive() external payable {}
}
