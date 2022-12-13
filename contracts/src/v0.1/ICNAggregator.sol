//SPDX-License-Identifier: MIT
// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.4/Aggregator.sol
pragma solidity ^0.8.16;

import "./ICNClient.sol";
import "./libraries/Math.sol";
import "./interfaces/IAggregator.sol";

/**
 * @title onChain Aggregation Contract
 * @notice Runs onChain aggregation recieving answers from multiple nodes
 */

contract ICNAggregator is ICNClient, IAggregator {
    struct Answer {
        uint128 minimumResponses;
        uint128 maxResponses;
        int256[] responses;
    }

    int256 private currentAnswerValue;
    uint256 private latestCompletedAnswer;
    uint256 private updatedTimestampValue;
    uint256 private constant MAX_ORACLE_COUNT = 10;
    uint256 private answerCounter = 1;
    uint128 public minimumResponses;
    bytes32[] public jobIds;
    address[] public oracles;

    mapping(address => bool) public authorizedRequesters;
    mapping(bytes32 => uint256) private requestAnswers;
    mapping(uint256 => Answer) private answers;
    mapping(uint256 => int256) private currentAnswers;
    mapping(uint256 => uint256) private updatedTimestamps;

    event NewRound(uint256 answerCounter, address aggregatorAddress, uint256 roundTimestamp);
    event ResponseRecieved(int256 indexed response, uint256 indexed answerId, address indexed sender);
    event AnswerUpdated(int256 _answerResponse, uint256 _answerId, uint256 dateAnswerUpdated);

    /**
     * @notice - Validating if oracles length match requirements of maximum and minimum
     */
    modifier validateRequestInfo(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds) {
        require(_oracles.length <= MAX_ORACLE_COUNT, "Cannot exceed max oracles");
        require(
            _oracles.length >= _minimumResponses, "Must have atleast minimum amount of oracles to obtain min responses"
        );
        require(_oracles.length == _jobIds.length, "Must have same amount of oracles as jobIds");
        _;
    }

    /**
     * @dev Prevents taking an action if not all responses are received for an answer.
     * @param _answerId The the identifier of the answer that keeps track of the responses.
     */
    modifier ensureAllResponsesReceived(uint256 _answerId) {
        if (answers[_answerId].responses.length == answers[_answerId].maxResponses) {
            _;
        }
    }

    /**
     * @notice Prevents taking an action if minimum number of responses has not been recieved
     */
    modifier ensureMinResponsesRecieved(uint256 _answerId) {
        if (answers[_answerId].responses.length >= answers[_answerId].minimumResponses) {
            _;
        }
    }

    /**
     * @notice Prevents taking an action if a newer answer has been recorded
     */
    modifier ensureOnlyLatestAnswer(uint256 _answerId) {
        if (latestCompletedAnswer <= _answerId) {
            _;
        }
    }

    /**
     * @notice Deploy contract with array of minimum responses, oracles and jobIds
     */
    constructor(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds) {
        updateRequestDetails(_minimumResponses, _oracles, _jobIds);
    }

    /**
     * @notice Updates the arrays of oracles and JobIds with new values
     */
    function updateRequestDetails(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds)
        public
        validateRequestInfo(_minimumResponses, _oracles, _jobIds)
    {
        minimumResponses = _minimumResponses;
        oracles = _oracles;
        jobIds = _jobIds;
    }

    /**
     * @notice requestRate - Creates an ICN Request for each oracle in the oracle array
     *
     */
    function requestRate() external {
        ICN.Request memory request;
        bytes32 requestId;

        for (uint256 i; i < oracles.length; i++) {
            request = buildRequest(jobIds[i], address(this), this.ICNCallback.selector);
            requestId = sendRequestTo(oracles[i], request);
            requestAnswers[requestId] = answerCounter;
        }

        answers[answerCounter].minimumResponses = minimumResponses;
        answers[answerCounter].maxResponses = uint128(oracles.length);

        emit NewRound(answerCounter, msg.sender, block.timestamp);

        answerCounter++;
    }

    /**
     * @notice Recieves the answer from the ICN Node
     * @dev this function can only called by the ICN Node Oracle that recieved the request
     */
    function ICNCallback(bytes32 _requestId, int256 _response) external {
        validateCallback(_requestId);
        uint256 answerId = requestAnswers[_requestId];
        delete requestAnswers[_requestId];

        answers[answerId].responses.push(_response);
        emit ResponseRecieved(_response, answerId, msg.sender);
        updateLatestAnswer(answerId);
        deleteAnswer(answerId);
    }

    /**
     * @notice Performs aggregation of the answers recieved from the ICN Node
     * Assuming atleast half of the oracles are honest.
     */
    function updateLatestAnswer(uint256 _answerId)
        private
        ensureMinResponsesRecieved(_answerId)
        ensureOnlyLatestAnswer(_answerId)
    {
        uint256 responseLength = answers[_answerId].responses.length;
        uint256 middleIndex = responseLength / 2;
        int256 currentAnswerTemp;
        if (responseLength % 2 == 0) {
            // SUM OF MEDIAN ALGO USED BY CHAINLINK - TEST FAILS FOR THIS - INTENDED ANSWER NOT RETURNED
            // int256 median1 = Math.quickselect(answers[_answerId].responses, middleIndex);
            // int256 median2 = Math.quickselect(answers[_answerId].responses, middleIndex + 1);
            // currentAnswerTemp = median1 + median2 / 2;
            //////////
            currentAnswerTemp = Math.quickselect(answers[_answerId].responses, middleIndex);
        } else {
            currentAnswerTemp = Math.quickselect(answers[_answerId].responses, middleIndex + 1);
        }
        currentAnswerValue = currentAnswerTemp;
        latestCompletedAnswer = _answerId;
        updatedTimestampValue = block.timestamp;
        updatedTimestamps[_answerId] = block.timestamp;
        currentAnswers[_answerId] = currentAnswerTemp;
        emit AnswerUpdated(currentAnswerTemp, _answerId, block.timestamp);
    }

    /**
     * @dev Cleans up the answer record if all responses have been received.
     * @param _answerId The identifier of the answer to be deleted
     */
    function deleteAnswer(uint256 _answerId) private ensureAllResponsesReceived(_answerId) {
        delete answers[_answerId];
    }

    ///// GETTERS //////

    /**
     * @notice get the most recently reported answer
     */
    function getlatestAnswer() external view returns (int256) {
        return currentAnswers[latestCompletedAnswer];
    }

    /**
     * @notice get the last updated at block timestamp
     */
    function getlatestTimestamp() external view returns (uint256) {
        return updatedTimestamps[latestCompletedAnswer];
    }

    /**
     * @notice get past rounds answers
     * @param _roundId the answer number to retrieve the answer for
     */
    function getAnswer(uint256 _roundId) external view returns (int256) {
        return currentAnswers[_roundId];
    }

    /**
     * @notice get block timestamp when an answer was last updated
     * @param _roundId the answer number to retrieve the updated timestamp for
     */
    function getTimestamp(uint256 _roundId) external view returns (uint256) {
        return updatedTimestamps[_roundId];
    }

    /**
     * @notice get the latest completed round where the answer was updated
     */
    function getlatestRound() external view returns (uint256) {
        return latestCompletedAnswer;
    }
}
