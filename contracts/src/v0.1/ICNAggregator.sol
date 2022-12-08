//SPDX-License-Identifier: MIT

pragma solidity ^0.8.16;

/**
 * @title onChain Aggregation Contract
 * @notice Runs onChain aggregation recieving answers from multiple nodes
 */

contract ICNAggregator {
    struct Answer {
        uint128 minimumResponses;
        uint128 maxResponses;
        uint256[] responses;
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

    event ResponseRecieved(int256 indexed response, uint256 indexed answerId, address indexed sender);

    /**
     * @notice - Validating if oracles length match requirements of maximum and minimum
     */
    modifier validateAnswer(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds) {
        require(_oracles.length <= MAX_ORACLE_COUNT, "Cannot exceed max oracles");
        require(
            _oracles.length >= _minimumResponses, "Must have atleast minimum amount of oracles to obtain min responses"
        );
        require(_oracles.length == _jobIds.length, "Must have same amount of oracles as jobIds");
        _;
    }

    /**
     * @notice Deploy contract with array of minimum responses, oracles and jobIds
     */
    constructor(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds) public {
        updateRequestDetails(_minimumResponses, _oracles, _jobIds);
    }

    /**
     * @notice Updates the arrays of oracles and JobIds with new values
     */
    function updateRequestDetails(uint128 _minimumResponses, address[] memory _oracles, bytes32[] memory _jobIds)
        public
        validateAnswer(_minimumResponses, _oracles, _jobIds)
    {
        minimumResponses = _minimumResponses;
        oracles = _oracles;
        jobIds = _jobIds;
    }
}
