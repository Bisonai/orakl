// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IL2Aggregator.sol";

contract L2Endpoint is Ownable {
    uint256 public aggregatorCount;
    uint256 public submitterCount;
    uint64 sNonce;

    mapping(address => bool) submitters;
    mapping(address => bool) aggregators;

    error InvalidSubmitter(address submitter);
    error InvalidAggregator(address aggregator);

    event SubmitterAdded(address newSubmitter);
    event SubmitterRemoved(address newSubmitter);
    event AggregatorAdded(address newAggregator);
    event AggregatorRemoved(address newAggregator);
    event Submitted(uint256 roundId, int256 submission);
    event RandomWordsRequested(
        bytes32 indexed keyHash,
        uint256 requestId,
        uint256 preSeed,
        uint64 indexed accId,
        uint32 callbackGasLimit,
        uint32 numWords,
        address indexed sender,
        bool isDirectPayment
    );

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

    function computeRequestId(
        bytes32 keyHash,
        address sender,
        uint64 accId,
        uint64 nonce
    ) private pure returns (uint256, uint256) {
        uint256 preSeed = uint256(keccak256(abi.encode(keyHash, sender, accId, nonce)));
        uint256 requestId = uint256(keccak256(abi.encode(keyHash, preSeed)));
        return (requestId, preSeed);
    }

    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords,
        bool isDirectPayment
    ) private returns (uint256) {
        sNonce++;
        (uint256 requestId, uint256 preSeed) = computeRequestId(keyHash, msg.sender, accId, sNonce);
        emit RandomWordsRequested(
            keyHash,
            requestId,
            preSeed,
            accId,
            callbackGasLimit,
            numWords,
            msg.sender,
            isDirectPayment
        );

        return requestId;
    }

    function requestRandomWords(
        bytes32 keyHash,
        uint64 accId,
        uint32 callbackGasLimit,
        uint32 numWords
    ) external returns (uint256) {
        bool isDirectPayment = false;
        uint256 requestId = requestRandomWords(
            keyHash,
            accId,
            callbackGasLimit,
            numWords,
            isDirectPayment
        );

        return requestId;
    }
}
