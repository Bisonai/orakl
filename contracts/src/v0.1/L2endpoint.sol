// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;
import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IL2Aggregator.sol";
import "./VRFConsumerBase.sol";

contract L2Endpoint is Ownable {
    uint256 private constant GAS_FOR_CALL_EXACT_CHECK = 5_000;
    uint256 public aggregatorCount;
    uint256 public submitterCount;
    uint64 sNonce;
    struct RequestInfo {
        address owner;
        uint32 callbackGasLimit;
    }
    mapping(address => bool) submitters;
    mapping(address => bool) aggregators;

    mapping(uint256 => RequestInfo) internal sRequestDetail;

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
    event RandomWordsFulfilled(uint256 indexed requestId, uint256[] randomWords, bool success);

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
            msg.sender,
            isDirectPayment
        );

        return requestId;
    }

    /**
     * @dev calls target address with exactly gasAmount gas and data as calldata
     * or reverts if at least gasAmount gas is not available.
     */
    function callWithExactGas(
        uint256 gasAmount,
        address target,
        bytes memory data
    ) internal returns (bool success) {
        // solhint-disable-next-line no-inline-assembly
        assembly {
            let g := gas()
            // Compute g -= GAS_FOR_CALL_EXACT_CHECK and check for underflow
            // The gas actually passed to the callee is min(gasAmount, 63//64*gas available).
            // We want to ensure that we revert if gasAmount >  63//64*gas available
            // as we do not want to provide them with less, however that check itself costs
            // gas.  GAS_FOR_CALL_EXACT_CHECK ensures we have at least enough gas to be able
            // to revert if gasAmount >  63//64*gas available.
            if lt(g, GAS_FOR_CALL_EXACT_CHECK) {
                revert(0, 0)
            }
            g := sub(g, GAS_FOR_CALL_EXACT_CHECK)
            // if g - g//64 <= gasAmount, revert
            // (we subtract g//64 because of EIP-150)
            if iszero(gt(sub(g, div(g, 64)), gasAmount)) {
                revert(0, 0)
            }
            // solidity calls check that a contract actually exists at the destination, so we do the same
            if iszero(extcodesize(target)) {
                revert(0, 0)
            }
            // call and return whether we succeeded. ignore return data
            // call(gas,addr,value,argsOffset,argsLength,retOffset,retLength)
            success := call(gasAmount, target, 0, add(data, 0x20), mload(data), 0, 0)
        }
        return success;
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

    function fulfillRandomWords(uint256 requestId, uint256[] memory randomWords) external {
        RequestInfo memory detail = sRequestDetail[requestId];
        bytes memory resp = abi.encodeWithSelector(
            VRFConsumerBase.rawFulfillRandomWords.selector,
            requestId,
            randomWords
        );
        bool success = callWithExactGas(detail.callbackGasLimit, detail.owner, resp);
        emit RandomWordsFulfilled(requestId, randomWords, success);
    }
}
