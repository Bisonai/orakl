// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.6/FluxAggregator.sol

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/ITypeAndVersion.sol";
import "./interfaces/IAggregator.sol";
import "./interfaces/IAggregatorValidator.sol";
import "./libraries/Median.sol";

/// @title Orakl Network Aggregator
/// @notice Handles aggregating data pushed in from off-chain. Oracles'
/// submissions are gathered in rounds, with each round aggregating the
/// submissions for each oracle into a single answer. The latest
/// aggregated answer is exposed as well as historical answers and
/// their updated at timestamp.
contract Aggregator is Ownable, IAggregator, ITypeAndVersion {
    struct Round {
        int256 answer;
        uint64 startedAt;
        uint64 updatedAt;
        uint32 answeredInRound;
    }

    struct RoundDetails {
        int256[] submissions;
        uint32 maxSubmissions;
        uint32 minSubmissions;
        uint32 timeout;
    }

    struct OracleStatus {
        uint32 startingRound;
        uint32 endingRound;
        uint32 lastReportedRound;
        uint32 lastStartedRound;
        int256 latestSubmission;
        uint16 index;
    }

    struct Requester {
        bool authorized;
        uint32 delay;
        uint32 lastStartedRound;
    }

    IAggregatorValidator public validator;

    // Round related params
    uint32 public maxSubmissionCount;
    uint32 public minSubmissionCount;
    uint32 public restartDelay;
    uint32 public timeout;
    uint8 public override decimals;
    string public override description;

    uint256 public constant MAX_ORACLE_COUNT = 77;
    uint32 private constant ROUND_MAX = 2 ** 32 - 1;
    uint256 private constant VALIDATOR_GAS_LIMIT = 100000;

    uint32 private reportingRoundId;
    uint32 internal latestRoundId;
    mapping(address => OracleStatus) private oracles;
    mapping(uint32 => Round) internal rounds;
    mapping(uint32 => RoundDetails) internal details;
    mapping(address => Requester) internal requesters;
    address[] private oracleAddresses;

    error OracleAlreadyEnabled();
    error OracleNotEnabled();
    error OffChainReadingOnly();
    error RequesterNotAuthorized();
    error PrevRoundNotSupersedable();
    error RoundNotAcceptingSubmission();
    error TooManyOracles();
    error NoDataPresent();
    error NewRequestTooSoon();
    error MinSubmissionGtMaxSubmission();
    error RestartDelayExceedOracleNum();
    error MinSubmissionZero();
    error MaxSubmissionGtOracleNum();

    event RoundDetailsUpdated(
        uint32 indexed minSubmissionCount,
        uint32 indexed maxSubmissionCount,
        uint32 restartDelay,
        uint32 timeout // measured in seconds
    );
    event OraclePermissionsUpdated(address indexed oracle, bool indexed whitelisted);
    event SubmissionReceived(
        int256 indexed submission,
        uint32 indexed round,
        address indexed oracle
    );
    event RequesterPermissionsSet(address indexed requester, bool authorized, uint32 delay);
    event ValidatorUpdated(address indexed previous, address indexed current);

    /**
     * @notice set up the aggregator with initial configuration
     * @param _timeout is the number of seconds after the previous round that are
     * allowed to lapse before allowing an oracle to skip an unfinished round
     * @param _validator is an optional contract address for validating
     * external validation of answers
     * @param _decimals represents the number of decimals to offset the answer by
     * @param _description a short description of what is being reported
     */
    constructor(uint32 _timeout, address _validator, uint8 _decimals, string memory _description) {
        updateFutureRounds(0, 0, 0, _timeout);
        setValidator(_validator);
        decimals = _decimals;
        description = _description;

        rounds[0].updatedAt = uint64(block.timestamp - uint256(_timeout));
    }

    /**
     * @notice called by oracles when they have witnessed a need to update
     * @param _roundId is the ID of the round this submission pertains to
     * @param _submission is the updated data that the oracle is submitting
     */
    function submit(uint256 _roundId, int256 _submission) external {
        bytes memory error = validateOracleRound(msg.sender, uint32(_roundId));
        require(error.length == 0, string(error));

        oracleInitializeNewRound(uint32(_roundId));
        recordSubmission(_submission, uint32(_roundId));
        (bool updated, int256 newAnswer) = updateRoundAnswer(uint32(_roundId));
        deleteRoundDetails(uint32(_roundId));
        if (updated) {
            validateAnswer(uint32(_roundId), newAnswer);
        }
    }

    /**
     * @notice called by the owner to remove and add new oracles as well as
     * update the round related parameters that pertain to total oracle count
     * @param _removed is the list of addresses for the new Oracles being removed
     * @param _added is the list of addresses for the new Oracles being added
     * @param _minSubmissionCount is the new minimum submission count for each round
     * @param _maxSubmissionCount is the new maximum submission count for each round
     * @param _restartDelay is the number of rounds an Oracle has to wait before
     * they can initiate a round
     */
    function changeOracles(
        address[] calldata _removed,
        address[] calldata _added,
        uint32 _minSubmissionCount,
        uint32 _maxSubmissionCount,
        uint32 _restartDelay
    ) external onlyOwner {
        for (uint256 i = 0; i < _removed.length; i++) {
            removeOracle(_removed[i]);
        }

        if (uint256(oracleCount()) + _added.length >= MAX_ORACLE_COUNT) {
            revert TooManyOracles();
        }

        for (uint256 i = 0; i < _added.length; i++) {
            addOracle(_added[i]);
        }

        updateFutureRounds(_minSubmissionCount, _maxSubmissionCount, _restartDelay, timeout);
    }

    /**
     * @notice update the round and payment related parameters for subsequent
     * rounds
     * @param _minSubmissionCount is the new minimum submission count for each round
     * @param _maxSubmissionCount is the new maximum submission count for each round
     * @param _restartDelay is the number of rounds an Oracle has to wait before
     * they can initiate a round
     */
    function updateFutureRounds(
        uint32 _minSubmissionCount,
        uint32 _maxSubmissionCount,
        uint32 _restartDelay,
        uint32 _timeout
    ) public onlyOwner {
        uint32 oracleNum = oracleCount(); // Save on storage reads

        if (_minSubmissionCount > _maxSubmissionCount) {
            revert MinSubmissionGtMaxSubmission();
        }

        if (_maxSubmissionCount > oracleNum) {
            revert MaxSubmissionGtOracleNum();
        }

        if (oracleNum > 0) {
            if (oracleNum <= _restartDelay) {
                revert RestartDelayExceedOracleNum();
            }
            if (_minSubmissionCount == 0) {
                revert MinSubmissionZero();
            }
        }

        minSubmissionCount = _minSubmissionCount;
        maxSubmissionCount = _maxSubmissionCount;
        restartDelay = _restartDelay;
        timeout = _timeout;

        emit RoundDetailsUpdated(_minSubmissionCount, _maxSubmissionCount, _restartDelay, _timeout);
    }

    /**
     * @notice returns the number of oracles
     */
    function oracleCount() public view returns (uint8) {
        return uint8(oracleAddresses.length);
    }

    /**
     * @notice returns an array of addresses containing the oracles on contract
     */
    function getOracles() external view returns (address[] memory) {
        return oracleAddresses;
    }

    /**
     * @notice get data about a round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * @param _roundId the round ID to retrieve the round data for
     * @return roundId is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started. This is 0
     * if the round hasn't been started yet.
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed. answeredInRound may be smaller than roundId when the round
     * timed out. answeredInRound is equal to roundId when the round didn't time out
     * and was completed regularly.
     * @dev Note that for in-progress rounds (i.e. rounds that haven't yet received
     * maxSubmissions) answer and updatedAt may change between queries.
     */
    function getRoundData(
        uint80 _roundId
    )
        public
        view
        virtual
        override
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        Round memory r = rounds[uint32(_roundId)];

        if (r.answeredInRound == 0 || !validRoundId(_roundId)) {
            revert NoDataPresent();
        }

        return (_roundId, r.answer, r.startedAt, r.updatedAt, r.answeredInRound);
    }

    /**
     * @notice get data about the latest round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * @return roundId is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started. This is 0
     * if the round hasn't been started yet.
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed. answeredInRound may be smaller than roundId when the round
     * timed out. answeredInRound is equal to roundId when the round didn't time
     * out and was completed regularly.
     * @dev Note that for in-progress rounds (i.e. rounds that haven't yet
     * received maxSubmissions) answer and updatedAt may change between queries.
     */
    function latestRoundData()
        public
        view
        virtual
        override
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return getRoundData(latestRoundId);
    }

    /**
     * @notice allows non-oracles to request a new round
     */
    function requestNewRound() external returns (uint80) {
        if (!requesters[msg.sender].authorized) {
            revert RequesterNotAuthorized();
        }

        uint32 current = reportingRoundId;
        if (rounds[current].updatedAt == 0 && !timedOut(current)) {
            revert PrevRoundNotSupersedable();
        }

        uint32 newRoundId = current + 1;
        requesterInitializeNewRound(newRoundId);
        return newRoundId;
    }

    /**
     * @notice allows the owner to specify new non-oracles to start new rounds
     * @param _requester is the address to set permissions for
     * @param _authorized is a boolean specifying whether they can start new rounds or not
     * @param _delay is the number of rounds the requester must wait before starting another round
     */
    function setRequesterPermissions(
        address _requester,
        bool _authorized,
        uint32 _delay
    ) external onlyOwner {
        if (requesters[_requester].authorized == _authorized) {
            return;
        }

        if (_authorized) {
            requesters[_requester].authorized = _authorized;
            requesters[_requester].delay = _delay;
        } else {
            delete requesters[_requester];
        }

        emit RequesterPermissionsSet(_requester, _authorized, _delay);
    }

    /**
     * @notice a method to provide all current info oracles need. Intended only
     * only to be callable by oracles. Not for use by contracts to read state.
     * @param _oracle the address to look up information for.
     */
    function oracleRoundState(
        address _oracle,
        uint32 _queriedRoundId
    )
        external
        view
        returns (
            bool _eligibleToSubmit,
            uint32 _roundId,
            int256 _latestSubmission,
            uint64 _startedAt,
            uint64 _timeout,
            uint8 _oracleCount
        )
    {
        if (msg.sender != tx.origin) {
            revert OffChainReadingOnly();
        }

        if (_queriedRoundId > 0) {
            Round storage round = rounds[_queriedRoundId];
            RoundDetails storage _details = details[_queriedRoundId];
            return (
                eligibleForSpecificRound(_oracle, _queriedRoundId),
                _queriedRoundId,
                oracles[_oracle].latestSubmission,
                round.startedAt,
                _details.timeout,
                oracleCount()
            );
        } else {
            return oracleRoundStateSuggestRound(_oracle);
        }
    }

    function currentRoundStartedAt() external view returns (uint256) {
        Round storage round = rounds[reportingRoundId];
        return round.startedAt;
    }

    /**
     * @notice method to update the address which does external data validation.
     * @param _newValidator designates the address of the new validation contract.
     */
    function setValidator(address _newValidator) public onlyOwner {
        address previous = address(validator);

        if (previous != _newValidator) {
            validator = IAggregatorValidator(_newValidator);
            emit ValidatorUpdated(previous, _newValidator);
        }
    }

    /**
     * @notice The type and version of this contract
     * @return Type and version string
     */
    function typeAndVersion() external pure virtual override returns (string memory) {
        return "Aggregator v0.1";
    }

    /**
     * Private
     */

    /**
     * @dev The function is executed fully with a specified parameter
     * at most once.
     */
    function initializeNewRound(uint32 _roundId) private {
        updateTimedOutRoundInfo(_roundId - 1);

        reportingRoundId = _roundId;
        RoundDetails memory nextDetails = RoundDetails(
            new int256[](0),
            maxSubmissionCount,
            minSubmissionCount,
            timeout
        );
        details[_roundId] = nextDetails;
        rounds[_roundId].startedAt = uint64(block.timestamp);

        emit NewRound(_roundId, msg.sender, rounds[_roundId].startedAt);
    }

    /**
     * @dev The function is executed fully with a specified parameter
     * at most once. Update of `reportingRoundId` is performed within
     * `initializeNewRound` which is used in `newRound()` function to
     * restrict access to this function.
     */
    function oracleInitializeNewRound(uint32 _roundId) private {
        if (!newRound(_roundId)) {
            return;
        }
        uint256 lastStarted = oracles[msg.sender].lastStartedRound;
        if (lastStarted > 0 && _roundId <= lastStarted + restartDelay) {
            return;
        }

        initializeNewRound(_roundId);
        oracles[msg.sender].lastStartedRound = _roundId;
    }

    function requesterInitializeNewRound(uint32 _roundId) private {
        if (!newRound(_roundId)) {
            return;
        }
        uint256 lastStarted = requesters[msg.sender].lastStartedRound;
        if (lastStarted > 0 && _roundId <= lastStarted + requesters[msg.sender].delay) {
            revert NewRequestTooSoon();
        }

        initializeNewRound(_roundId);
        requesters[msg.sender].lastStartedRound = _roundId;
    }

    function updateTimedOutRoundInfo(uint32 _roundId) private {
        if (!timedOut(_roundId)) {
            return;
        }

        uint32 prevId = _roundId - 1;
        rounds[_roundId].answer = rounds[prevId].answer;
        rounds[_roundId].answeredInRound = rounds[prevId].answeredInRound;
        rounds[_roundId].updatedAt = uint64(block.timestamp);

        delete details[_roundId];
    }

    function eligibleForSpecificRound(
        address _oracle,
        uint32 _queriedRoundId
    ) private view returns (bool _eligible) {
        if (rounds[_queriedRoundId].startedAt > 0) {
            // past or current round
            return
                acceptingSubmissions(_queriedRoundId) &&
                validateOracleRound(_oracle, _queriedRoundId).length == 0;
        } else {
            // future rounds
            return
                delayed(_oracle, _queriedRoundId) &&
                validateOracleRound(_oracle, _queriedRoundId).length == 0;
        }
    }

    function oracleRoundStateSuggestRound(
        address _oracle
    )
        private
        view
        returns (
            bool _eligibleToSubmit,
            uint32 _roundId,
            int256 _latestSubmission,
            uint64 _startedAt,
            uint64 _timeout,
            uint8 _oracleCount
        )
    {
        Round storage round = rounds[0];
        OracleStatus storage oracle = oracles[_oracle];

        bool shouldSupersede = oracle.lastReportedRound == reportingRoundId ||
            !acceptingSubmissions(reportingRoundId);

        // Instead of nudging oracles to submit to the next round, the
        // inclusion of the shouldSupersede bool in the if condition
        // pushes them towards submitting in a currently open round.
        if (supersedable(reportingRoundId) && shouldSupersede) {
            _roundId = reportingRoundId + 1;
            round = rounds[_roundId];
            _eligibleToSubmit = delayed(_oracle, _roundId);
        } else {
            _roundId = reportingRoundId;
            round = rounds[_roundId];
            _eligibleToSubmit = acceptingSubmissions(_roundId);
        }

        if (validateOracleRound(_oracle, _roundId).length != 0) {
            _eligibleToSubmit = false;
        }

        return (
            _eligibleToSubmit,
            _roundId,
            oracle.latestSubmission,
            round.startedAt,
            details[_roundId].timeout,
            oracleCount()
        );
    }

    function updateRoundAnswer(uint32 _roundId) internal returns (bool, int256) {
        if (details[_roundId].submissions.length < details[_roundId].minSubmissions) {
            return (false, 0);
        }

        int256 newAnswer = Median.calculateInplace(details[_roundId].submissions);
        rounds[_roundId].answer = newAnswer;
        rounds[_roundId].updatedAt = uint64(block.timestamp);
        rounds[_roundId].answeredInRound = _roundId;
        latestRoundId = _roundId;

        emit AnswerUpdated(newAnswer, _roundId, block.timestamp);

        return (true, newAnswer);
    }

    function validateAnswer(uint32 _roundId, int256 _newAnswer) private {
        IAggregatorValidator av = validator;
        if (address(av) == address(0)) {
            return;
        }

        uint32 prevRound = _roundId - 1;
        uint32 prevAnswerRoundId = rounds[prevRound].answeredInRound;
        int256 prevRoundAnswer = rounds[prevRound].answer;
        // We do not want the validator to ever prevent reporting, so we limit its
        // gas usage and catch any errors that may arise.
        try
            av.validate{gas: VALIDATOR_GAS_LIMIT}(
                prevAnswerRoundId,
                prevRoundAnswer,
                _roundId,
                _newAnswer
            )
        {} catch {}
    }

    function recordSubmission(int256 _submission, uint32 _roundId) private {
        if (!acceptingSubmissions(_roundId)) {
            revert RoundNotAcceptingSubmission();
        }

        details[_roundId].submissions.push(_submission);
        oracles[msg.sender].lastReportedRound = _roundId;
        oracles[msg.sender].latestSubmission = _submission;

        emit SubmissionReceived(_submission, _roundId, msg.sender);
    }

    function deleteRoundDetails(uint32 _roundId) private {
        if (details[_roundId].submissions.length >= details[_roundId].maxSubmissions) {
            delete details[_roundId];
        }
    }

    function timedOut(uint32 _roundId) private view returns (bool) {
        uint64 startedAt = rounds[_roundId].startedAt;
        // `details` mapping for a specific round is supposed to be
        // deleted after aggregator collects at least `maxSubmissions`
        // answers for that specific round. If it does not collect
        // such number of answers within the round, `details` mapping
        // will be deleted during the initialization of the next
        // round.  Depending on the validity of `_roundId` key,
        // `roundTimeout` variable can be used to distinguish between
        // successfully acomplished round and unsuccesful
        // one. Assuming that `roundId` has been active (startedAt >
        // 0), and a `roundTimeout` is over, a non-negative value of
        // `roundTimeout` represents unsuccessfully finished round.
        uint32 roundTimeout = details[_roundId].timeout;
        return startedAt > 0 && roundTimeout > 0 && startedAt + roundTimeout < block.timestamp;
    }

    function getStartingRound(address _oracle) private view returns (uint32) {
        uint32 currentRound = reportingRoundId;
        if (currentRound != 0 && currentRound == oracles[_oracle].endingRound) {
            return currentRound;
        }
        return currentRound + 1;
    }

    /**
     * @dev In general, oracles are supposed to submit to the current
     * round that is denoted by `reportingRoundId` variable. In some
     * cases, when the current round is `supersedable`, we allow them to
     * submit to the next round.
     * The only exception of submitting on previous round
     * (`reportingRound - 1`) is when the current round has not
     * received enough submissions to produce an aggregate value.
     */
    function previousAndCurrentUnanswered(
        uint32 _roundId,
        uint32 _rrId
    ) private view returns (bool) {
        return _roundId + 1 == _rrId && rounds[_rrId].updatedAt == 0;
    }

    function addOracle(address _oracle) private {
        if (oracleEnabled(_oracle)) {
            revert OracleAlreadyEnabled();
        }
        oracles[_oracle].startingRound = getStartingRound(_oracle);
        oracles[_oracle].endingRound = ROUND_MAX;
        oracles[_oracle].index = uint16(oracleAddresses.length);
        oracleAddresses.push(_oracle);

        emit OraclePermissionsUpdated(_oracle, true);
    }

    function removeOracle(address _oracle) private {
        if (!oracleEnabled(_oracle)) {
            revert OracleNotEnabled();
        }
        oracles[_oracle].endingRound = reportingRoundId + 1;
        address tail = oracleAddresses[uint256(oracleCount()) - 1];
        uint16 index = oracles[_oracle].index;
        oracles[tail].index = index;
        delete oracles[_oracle].index;
        oracleAddresses[index] = tail;
        oracleAddresses.pop();

        emit OraclePermissionsUpdated(_oracle, false);
    }

    function validateOracleRound(
        address _oracle,
        uint32 _roundId
    ) private view returns (bytes memory) {
        uint32 startingRound = oracles[_oracle].startingRound;
        uint32 rrId = reportingRoundId;

        if (startingRound == 0) return "not enabled oracle";
        if (startingRound > _roundId) return "not yet enabled oracle";
        if (oracles[_oracle].endingRound < _roundId) return "no longer allowed oracle";
        if (oracles[_oracle].lastReportedRound >= _roundId)
            return "cannot report on previous rounds";
        if (
            // Not reporting on current round.
            _roundId != rrId &&
            // Not reporting on next round.
            _roundId != rrId + 1 &&
            // Not reporting on the previous round while the current
            // round has not finished yet.
            !previousAndCurrentUnanswered(_roundId, rrId)
        ) return "invalid round to report";
        if (_roundId != 1 && !supersedable(_roundId - 1)) return "previous round not supersedable";

        return "";
    }

    /**
     * @dev `updatedAt > 0` expression represents that aggregated
     * submission can be accessed through `AggregatorProxy`. Accepting
     * new submissions can still be allowed, but the aggregated value
     * should be already a good approximate, therefore the `_roundId`
     * is considered supersedable. The other possibility is that
     * previous current round has not received enough submissions to
     * compute aggregate value, however, the round has timed out,
     * therefore `_roundId` is supersedable.
     */
    function supersedable(uint32 _roundId) private view returns (bool) {
        return rounds[_roundId].updatedAt > 0 || timedOut(_roundId);
    }

    /**
     * @dev `endingRound` is set to `ROUND_MAX` when a new oracle is
     * added (`addOracle`), therefore the `endingRound` equality to
     * `ROUND_MAX` means that oracle has already been enabled.
     * When oracle is removed (`removeOracle`), `endingRound` property
     * is set to `reportingRoundId + 1`, indicating that the oracle
     * is diabled. Submitting to the current `reportingRoundId` is
     * still allowed.
     */
    function oracleEnabled(address _oracle) private view returns (bool) {
        return oracles[_oracle].endingRound == ROUND_MAX;
    }

    /**
     * @dev `maxSubmissions` struct property is initialized within
     * `initializeNewRound` function with uint32 storage variable
     * `maxSubmissionsCount`. After Aggregator collects at least
     * `maxSubmissions` submissions, `details` struct is deleted
     * (`deleteRoundDetails`), making `maxSubmissions` effectively
     * 0. This sequence of events is depended on in this
     * `acceptingSubmissions` to find out whether aggregator still
     * accepts submissions for `_roundId`.
     */
    function acceptingSubmissions(uint32 _roundId) private view returns (bool) {
        return details[_roundId].maxSubmissions != 0;
    }

    /**
     * @dev oracles can be limited by how frequently they can initiate
     * a new round. This frequency is defined with `restartDelay`
     * variable that is same for all oracles. When a new oracle is
     * added, it can initiate a new round without a
     * limitation. However, for later submissions it is constrained by
     * the frequency of new round initiation (`restartDelay`). Even
     * though oracle cannot initiate a new round, it can still submit
     * a new answer to aggregator. If `restartDelay` is 0, there are
     * no frequency limitations on initiating a new round.
     */
    function delayed(address _oracle, uint32 _roundId) private view returns (bool) {
        uint256 lastStarted = oracles[_oracle].lastStartedRound;
        return lastStarted == 0 || _roundId > lastStarted + restartDelay;
    }

    function newRound(uint32 _roundId) private view returns (bool) {
        return _roundId == reportingRoundId + 1;
    }

    function validRoundId(uint256 _roundId) private pure returns (bool) {
        return _roundId <= ROUND_MAX;
    }
}
