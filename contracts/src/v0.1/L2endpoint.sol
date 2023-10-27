// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IL2Aggregator.sol";

//test Git action
contract Endpoint is Ownable {
    uint256 public aggregatorCount;
    uint256 public submitterCount;

    mapping(address => bool) submitters;
    mapping(address => bool) aggregators;

    error InvalidSubmitter(address submitter);
    error InvalidAggregator(address aggregator);

    event SubmitterAdded(address newSubmitter);
    event SubmitterRemoved(address newSubmitter);
    event AggregatorAdded(address newAggregator);
    event AggregatorRemoved(address newAggregator);
    event Submitted(uint256 roundId, int256 submission);

    function addAggregator(address _newAggregator) external onlyOwner {
        if (aggregators[_newAggregator]) revert InvalidAggregator(_newAggregator);
        aggregators[_newAggregator] = true;
        aggregatorCount += 1;
        emit AggregatorAdded(_newAggregator);
    }

    function removeAggregator(address _aggregator) external onlyOwner {
        if (!aggregators[_aggregator]) revert InvalidAggregator(_aggregator);
        delete aggregators[_aggregator];
        aggregatorCount -= 1;
        emit AggregatorRemoved(_aggregator);
    }

    function addSubmitter(address _newSubmitter) external onlyOwner {
        if (submitters[_newSubmitter]) revert InvalidSubmitter(_newSubmitter);
        submitters[_newSubmitter] = true;
        submitterCount += 1;
        emit SubmitterAdded(_newSubmitter);
    }

    function removeSubmitter(address _submitter) external onlyOwner {
        if (!submitters[_submitter]) revert InvalidSubmitter(_submitter);
        delete submitters[_submitter];
        submitterCount -= 1;
        emit SubmitterRemoved(_submitter);
    }

    function submit(uint256 _roundId, int256 _submission, address _aggregator) external {
        if (!submitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        if (!aggregators[_aggregator]) revert InvalidAggregator(_aggregator);
        IL2Aggregator(_aggregator).submit(_roundId, _submission);
        emit Submitted(_roundId, _submission);
    }
}
