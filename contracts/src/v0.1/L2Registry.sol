// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";

contract Registry is Ownable {

    error NotEnoughFee();
    error InvalidChainID();
    
    event ChainProposed(address sender, L2Endpoint endPoint, uint chainID);
    event ChainConfirmed (uint chainID);
    event ProposeFeeSet(uint fee);

    uint public ProposeFee;
    uint public StartRound;

	struct L2Endpoint {
		string jsonRpc;
		address endpoint;
		address owner;
	}
	// chainId => L2 Endpoint
	mapping(uint256 => L2Endpoint) public chainRegistry;
    // pending proposal
    mapping (uint256 => L2Endpoint) pendingProposal;

  // L2 consumer => L1 payer
  // Can be updated only by Orakl Network through call from L2
  // mapping(address => address) accountRegistry;

	function proposeChain(uint chainID, L2Endpoint memory endpoint) external payable {
        if(msg.value<ProposeFee){
            revert NotEnoughFee();
        }
        pendingProposal[chainID] = endpoint;
        emit ChainProposed(msg.sender, endpoint, chainID);
	}

    function SetProposeFee(uint fee) public onlyOwner {
        ProposeFee = fee;
        emit ProposeFeeSet(fee);
    }

	function confirmChain(uint chainId) public onlyOwner {
        if (pendingProposal[chainId].owner == address(0))  {
            revert InvalidChainID();
        }
        chainRegistry[chainId] = pendingProposal[chainId];
        emit ChainConfirmed(chainId);
    }
}