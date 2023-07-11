// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IAggregator.sol";

contract Endpoint is Ownable {
    mapping(address => bool) submitters;

    error InvalidSubmitter(address submitter);
    event SubmitterAdded(address newSubmitter);
    event SubmitterRemoved(address newSubmitter);

    constructor(address _aggregator) {}

    function addSubmitter(address _newSubmitter) external onlyOwner {
        submitters[_newSubmitter] = true;
        emit SubmitterAdded(_newSubmitter);
    }

    function removeSubmitter(address _submitter) external onlyOwner {
        delete submitters[_submitter];
        emit SubmitterRemoved(_submitter);
    }

    function submit(uint256 _roundId, int256 _submission) external {
        if (!submitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        //update submission to aggregator
    }
}
