// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IL2Aggregator.sol";
import "./VRFConsumerBase.sol";
import "./L2EndpointBase.sol";
import "./L2EndpointRequestResponse.sol";

contract L2Endpoint is Ownable, L2EndpointRequestResponse {
    uint256 public sAggregatorCount;
    uint64 private sNonce;

    mapping(address => bool) sAggregators;

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
        address indexed sender
    );
    event RandomWordsFulfilled(uint256 indexed requestId, uint256[] randomWords, bool success);

    function addAggregator(address _newAggregator) external onlyOwner {
        if (sAggregators[_newAggregator]) revert InvalidAggregator(_newAggregator);
        sAggregators[_newAggregator] = true;
        sAggregatorCount += 1;
        emit AggregatorAdded(_newAggregator);
    }

    function removeAggregator(address _aggregator) external onlyOwner {
        if (!sAggregators[_aggregator]) revert InvalidAggregator(_aggregator);
        delete sAggregators[_aggregator];
        sAggregatorCount -= 1;
        emit AggregatorRemoved(_aggregator);
    }

    function addSubmitter(address _newSubmitter) external onlyOwner {
        if (sSubmitters[_newSubmitter]) revert InvalidSubmitter(_newSubmitter);
        sSubmitters[_newSubmitter] = true;
        sSubmitterCount += 1;
        emit SubmitterAdded(_newSubmitter);
    }

    function removeSubmitter(address _submitter) external onlyOwner {
        if (!sSubmitters[_submitter]) revert InvalidSubmitter(_submitter);
        delete sSubmitters[_submitter];
        sSubmitterCount -= 1;
        emit SubmitterRemoved(_submitter);
    }

    function submit(uint256 _roundId, int256 _submission, address _aggregator) external {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        if (!sAggregators[_aggregator]) revert InvalidAggregator(_aggregator);
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
        uint32 numWords
    ) external nonReentrant returns (uint256) {
        sNonce++;
        (uint256 requestId, uint256 preSeed) = computeRequestId(keyHash, msg.sender, accId, sNonce);
        sRequestDetail[requestId] = RequestInfo({
            owner: msg.sender,
            callbackGasLimit: callbackGasLimit
        });
        emit RandomWordsRequested(
            keyHash,
            requestId,
            preSeed,
            accId,
            callbackGasLimit,
            numWords,
            msg.sender
        );

        return requestId;
    }

    function fulfillRandomWords(
        uint256 requestId,
        uint256[] memory randomWords
    ) external nonReentrant {
        if (!sSubmitters[msg.sender]) revert InvalidSubmitter(msg.sender);
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            VRFConsumerBase.rawFulfillRandomWords.selector,
            requestId,
            randomWords
        );
        setReentrancy(true);
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        setReentrancy(false);
        emit RandomWordsFulfilled(requestId, randomWords, success);
    }
}
