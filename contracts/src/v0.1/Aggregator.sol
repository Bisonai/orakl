// SPDX-License-Identifier: MIT
pragma solidity 0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.6/FluxAggregator.sol

import "./Median.sol";
import "./ConfirmedOwner.sol";
import "./interfaces/AggregatorInterface.sol";
import "./interfaces/AggregatorValidatorInterface.sol";

/**
 * @title The Prepaid Aggregator contract
 * @notice Handles aggregating data pushed in from off-chain, and unlocks
 * payment for oracles as they report. Oracles' submissions are gathered in
 * rounds, with each round aggregating the submissions for each oracle into a
 * single answer. The latest aggregated answer is exposed as well as historical
 * answers and their updated at timestamp.
 */
contract Aggregator is AggregatorInterface, ConfirmedOwner {
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
        uint128 paymentAmount;
    }

    struct OracleStatus {
        uint128 withdrawable;
        uint32 startingRound;
        uint32 endingRound;
        uint32 lastReportedRound;
        uint32 lastStartedRound;
        int256 latestSubmission;
        uint16 index;
        address admin;
        address pendingAdmin;
    }

    struct Requester {
        bool authorized;
        uint32 delay;
        uint32 lastStartedRound;
    }

    struct Funds {
        uint128 available;
        uint128 allocated;
    }

    AggregatorValidatorInterface public validator;

    // Round related params
    uint128 public paymentAmount;
    uint32 public maxSubmissionCount;
    uint32 public minSubmissionCount;
    uint32 public restartDelay;
    uint32 public timeout;
    uint8 public override decimals;
    string public override description;

    int256 public immutable minSubmissionValue;
    int256 public immutable maxSubmissionValue;

    uint256 public constant override version = 3;

    /**
     * @notice To ensure owner isn't withdrawing required funds as oracles are
     * submitting updates, we enforce that the contract maintains a minimum
     * reserve of RESERVE_ROUNDS * oracleCount() KLAY earmarked for payment to
     * oracles. (Of course, this doesn't prevent the contract from running out of
     * funds without the owner's intervention.)
     */
    uint256 private constant RESERVE_ROUNDS = 2;
    uint256 private constant MAX_ORACLE_COUNT = 77;
    uint32 private constant ROUND_MAX = 2 ** 32 - 1;
    uint256 private constant VALIDATOR_GAS_LIMIT = 100000;
    // An error specific to the Aggregator V3 Interface, to prevent possible
    // confusion around accidentally reading unset values as reported values.
    string private constant V3_NO_DATA_ERROR = "No data present";

    uint32 private reportingRoundId;
    uint32 internal latestRoundId;
    mapping(address => OracleStatus) private oracles;
    mapping(uint32 => Round) internal rounds;
    mapping(uint32 => RoundDetails) internal details;
    mapping(address => Requester) internal requesters;
    address[] private oracleAddresses;
    Funds private recordedFunds;

    event AvailableFundsUpdated(uint256 indexed amount);
    event RoundDetailsUpdated(
        uint128 indexed paymentAmount,
        uint32 indexed minSubmissionCount,
        uint32 indexed maxSubmissionCount,
        uint32 restartDelay,
        uint32 timeout // measured in seconds
    );
    event OraclePermissionsUpdated(address indexed oracle, bool indexed whitelisted);
    event OracleAdminUpdated(address indexed oracle, address indexed newAdmin);
    event OracleAdminUpdateRequested(address indexed oracle, address admin, address newAdmin);
    event SubmissionReceived(
        int256 indexed submission,
        uint32 indexed round,
        address indexed oracle
    );
    event RequesterPermissionsSet(address indexed requester, bool authorized, uint32 delay);
    event ValidatorUpdated(address indexed previous, address indexed current);

    /**
     * @notice set up the aggregator with initial configuration
     * @param _paymentAmount The amount paid of KLAY paid to each oracle per submission, in peb (units of 10⁻¹⁸ KLAY)
     * @param _timeout is the number of seconds after the previous round that are
     * allowed to lapse before allowing an oracle to skip an unfinished round
     * @param _validator is an optional contract address for validating
     * external validation of answers
     * @param _minSubmissionValue is an immutable check for a lower bound of what
     * submission values are accepted from an oracle
     * @param _maxSubmissionValue is an immutable check for an upper bound of what
     * submission values are accepted from an oracle
     * @param _decimals represents the number of decimals to offset the answer by
     * @param _description a short description of what is being reported
     */
    constructor(
        uint128 _paymentAmount,
        uint32 _timeout,
        address _validator,
        int256 _minSubmissionValue,
        int256 _maxSubmissionValue,
        uint8 _decimals,
        string memory _description
    ) ConfirmedOwner(msg.sender) {
        updateFutureRounds(_paymentAmount, 0, 0, 0, _timeout);
        setValidator(_validator);
        minSubmissionValue = _minSubmissionValue;
        maxSubmissionValue = _maxSubmissionValue;
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
        require(_submission >= minSubmissionValue, "value below minSubmissionValue");
        require(_submission <= maxSubmissionValue, "value above maxSubmissionValue");
        require(error.length == 0, string(error));

        oracleInitializeNewRound(uint32(_roundId));
        recordSubmission(_submission, uint32(_roundId));
        (bool updated, int256 newAnswer) = updateRoundAnswer(uint32(_roundId));
        payOracle(uint32(_roundId));
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
     * @param _addedAdmins is the admin addresses for the new respective _added
     * list. Only this address is allowed to access the respective oracle's funds
     * @param _minSubmissionCount is the new minimum submission count for each round
     * @param _maxSubmissionCount is the new maximum submission count for each round
     * @param _restartDelay is the number of rounds an Oracle has to wait before
     * they can initiate a round
     */
    function changeOracles(
        address[] calldata _removed,
        address[] calldata _added,
        address[] calldata _addedAdmins,
        uint32 _minSubmissionCount,
        uint32 _maxSubmissionCount,
        uint32 _restartDelay
    ) external onlyOwner {
        for (uint256 i = 0; i < _removed.length; i++) {
            removeOracle(_removed[i]);
        }

        require(_added.length == _addedAdmins.length, "need same oracle and admin count");
        require(uint256(oracleCount()) + _added.length <= MAX_ORACLE_COUNT, "max oracles allowed");

        for (uint256 i = 0; i < _added.length; i++) {
            addOracle(_added[i], _addedAdmins[i]);
        }

        updateFutureRounds(
            paymentAmount,
            _minSubmissionCount,
            _maxSubmissionCount,
            _restartDelay,
            timeout
        );
    }

    /**
     * @notice update the round and payment related parameters for subsequent
     * rounds
     * @param _paymentAmount is the payment amount for subsequent rounds
     * @param _minSubmissionCount is the new minimum submission count for each round
     * @param _maxSubmissionCount is the new maximum submission count for each round
     * @param _restartDelay is the number of rounds an Oracle has to wait before
     * they can initiate a round
     */
    function updateFutureRounds(
        uint128 _paymentAmount,
        uint32 _minSubmissionCount,
        uint32 _maxSubmissionCount,
        uint32 _restartDelay,
        uint32 _timeout
    ) public onlyOwner {
        uint32 oracleNum = oracleCount(); // Save on storage reads
        require(_maxSubmissionCount >= _minSubmissionCount, "max must equal/exceed min");
        require(oracleNum >= _maxSubmissionCount, "max cannot exceed total");
        require(oracleNum == 0 || oracleNum > _restartDelay, "delay cannot exceed total");
        require(
            recordedFunds.available >= requiredReserve(_paymentAmount),
            "insufficient funds for payment"
        );
        if (oracleCount() > 0) {
            require(_minSubmissionCount > 0, "min must be greater than 0");
        }

        paymentAmount = _paymentAmount;
        minSubmissionCount = _minSubmissionCount;
        maxSubmissionCount = _maxSubmissionCount;
        restartDelay = _restartDelay;
        timeout = _timeout;

        emit RoundDetailsUpdated(
            paymentAmount,
            _minSubmissionCount,
            _maxSubmissionCount,
            _restartDelay,
            _timeout
        );
    }

    /**
     * @notice the amount of payment yet to be withdrawn by oracles
     */
    function allocatedFunds() external view returns (uint128) {
        return recordedFunds.allocated;
    }

    /**
     * @notice the amount of future funding available to oracles
     */
    function availableFunds() external view returns (uint128) {
        return recordedFunds.available;
    }

    /**
     * @notice recalculate the amount of KLAY available for payouts
     */
    function updateAvailableFunds() public {
        Funds memory funds = recordedFunds;

        uint256 nowAvailable = address(this).balance - funds.allocated;

        if (funds.available != nowAvailable) {
            recordedFunds.available = uint128(nowAvailable);
            emit AvailableFundsUpdated(nowAvailable);
        }
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

        require(r.answeredInRound > 0 && validRoundId(_roundId), V3_NO_DATA_ERROR);

        return (_roundId, r.answer, r.startedAt, r.updatedAt, r.answeredInRound);
    }

    /**
     * @notice get data about the latest round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values. Consumers are encouraged to
     * use this more fully featured method over the "legacy" latestRound/
     * latestAnswer/latestTimestamp functions. Consumers are encouraged to check
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
     * @notice query the available amount of KLAY for an oracle to withdraw
     */
    function withdrawablePayment(address _oracle) external view returns (uint256) {
        return oracles[_oracle].withdrawable;
    }

    /**
     * @notice transfers the oracle's KLAY to another address. Can only be called
     * by the oracle's admin.
     * @param _oracle is the oracle whose KLAY is transferred
     * @param _recipient is the address to send the KLAY to
     * @param _amount is the amount of KLAY to send
     */
    function withdrawPayment(address _oracle, address _recipient, uint256 _amount) external {
        require(oracles[_oracle].admin == msg.sender, "only callable by admin");

        // Safe to downcast _amount because the total amount of LINK is less than 2^128.
        uint128 amount = uint128(_amount);
        uint128 available = oracles[_oracle].withdrawable;
        require(available >= amount, "insufficient withdrawable funds");

        oracles[_oracle].withdrawable = available - amount;
        recordedFunds.allocated = recordedFunds.allocated - amount;

        /* assert(linkToken.transfer(_recipient, uint256(amount))); */
    }

    /**
     * @notice transfers the owner's KLAY to another address
     * @param _recipient is the address to send the KLAY to
     * @param _amount is the amount of KLAY to send
     */
    function withdrawFunds(address _recipient, uint256 _amount) external onlyOwner {
        uint256 available = uint256(recordedFunds.available);
        require(
            available - requiredReserve(paymentAmount) >= _amount,
            "insufficient reserve funds"
        );
        /* require(linkToken.transfer(_recipient, _amount), "token transfer failed"); */
        /* updateAvailableFunds(); */
    }

    /**
     * @notice get the admin address of an oracle
     * @param _oracle is the address of the oracle whose admin is being queried
     */
    function getAdmin(address _oracle) external view returns (address) {
        return oracles[_oracle].admin;
    }

    /**
     * @notice transfer the admin address for an oracle
     * @param _oracle is the address of the oracle whose admin is being transferred
     * @param _newAdmin is the new admin address
     */
    function transferAdmin(address _oracle, address _newAdmin) external {
        require(oracles[_oracle].admin == msg.sender, "only callable by admin");
        oracles[_oracle].pendingAdmin = _newAdmin;

        emit OracleAdminUpdateRequested(_oracle, msg.sender, _newAdmin);
    }

    /**
     * @notice accept the admin address transfer for an oracle
     * @param _oracle is the address of the oracle whose admin is being transferred
     */
    function acceptAdmin(address _oracle) external {
        require(oracles[_oracle].pendingAdmin == msg.sender, "only callable by pending admin");
        oracles[_oracle].pendingAdmin = address(0);
        oracles[_oracle].admin = msg.sender;

        emit OracleAdminUpdated(_oracle, msg.sender);
    }

    /**
     * @notice allows non-oracles to request a new round
     */
    function requestNewRound() external returns (uint80) {
        require(requesters[msg.sender].authorized, "not authorized requester");

        uint32 current = reportingRoundId;
        require(
            rounds[current].updatedAt > 0 || timedOut(current),
            "prev round must be supersedable"
        );

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
        if (requesters[_requester].authorized == _authorized) return;

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
            uint128 _availableFunds,
            uint8 _oracleCount,
            uint128 _paymentAmount
        )
    {
        require(msg.sender == tx.origin, "off-chain reading only");

        if (_queriedRoundId > 0) {
            Round storage round = rounds[_queriedRoundId];
            RoundDetails storage _details = details[_queriedRoundId];
            return (
                eligibleForSpecificRound(_oracle, _queriedRoundId),
                _queriedRoundId,
                oracles[_oracle].latestSubmission,
                round.startedAt,
                _details.timeout,
                recordedFunds.available,
                oracleCount(),
                (round.startedAt > 0 ? _details.paymentAmount : paymentAmount)
            );
        } else {
            return oracleRoundStateSuggestRound(_oracle);
        }
    }

    /**
     * @notice method to update the address which does external data validation.
     * @param _newValidator designates the address of the new validation contract.
     */
    function setValidator(address _newValidator) public onlyOwner {
        address previous = address(validator);

        if (previous != _newValidator) {
            validator = AggregatorValidatorInterface(_newValidator);

            emit ValidatorUpdated(previous, _newValidator);
        }
    }

    /**
     * @notice method to accept KLAY inside of contract
     */
    function deposit() public payable {
        updateAvailableFunds();
    }

    /**
     * Private
     */

    function initializeNewRound(uint32 _roundId) private {
        updateTimedOutRoundInfo(_roundId - 1);

        reportingRoundId = _roundId;
        RoundDetails memory nextDetails = RoundDetails(
            new int256[](0),
            maxSubmissionCount,
            minSubmissionCount,
            timeout,
            paymentAmount
        );
        details[_roundId] = nextDetails;
        rounds[_roundId].startedAt = uint64(block.timestamp);

        emit NewRound(_roundId, msg.sender, rounds[_roundId].startedAt);
    }

    function oracleInitializeNewRound(uint32 _roundId) private {
        if (!newRound(_roundId)) return;
        uint256 lastStarted = oracles[msg.sender].lastStartedRound; // cache storage reads
        if (_roundId <= lastStarted + restartDelay && lastStarted != 0) return;

        initializeNewRound(_roundId);

        oracles[msg.sender].lastStartedRound = _roundId;
    }

    function requesterInitializeNewRound(uint32 _roundId) private {
        if (!newRound(_roundId)) return;
        uint256 lastStarted = requesters[msg.sender].lastStartedRound; // cache storage reads
        require(
            _roundId > lastStarted + requesters[msg.sender].delay || lastStarted == 0,
            "must delay requests"
        );

        initializeNewRound(_roundId);

        requesters[msg.sender].lastStartedRound = _roundId;
    }

    function updateTimedOutRoundInfo(uint32 _roundId) private {
        if (!timedOut(_roundId)) return;

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
            return
                acceptingSubmissions(_queriedRoundId) &&
                validateOracleRound(_oracle, _queriedRoundId).length == 0;
        } else {
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
            uint128 _availableFunds,
            uint8 _oracleCount,
            uint128 _paymentAmount
        )
    {
        Round storage round = rounds[0];
        OracleStatus storage oracle = oracles[_oracle];

        bool shouldSupersede = oracle.lastReportedRound == reportingRoundId ||
            !acceptingSubmissions(reportingRoundId);
        // Instead of nudging oracles to submit to the next round, the inclusion of
        // the shouldSupersede bool in the if condition pushes them towards
        // submitting in a currently open round.
        if (supersedable(reportingRoundId) && shouldSupersede) {
            _roundId = reportingRoundId + 1;
            round = rounds[_roundId];

            _paymentAmount = paymentAmount;
            _eligibleToSubmit = delayed(_oracle, _roundId);
        } else {
            _roundId = reportingRoundId;
            round = rounds[_roundId];

            _paymentAmount = details[_roundId].paymentAmount;
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
            recordedFunds.available,
            oracleCount(),
            _paymentAmount
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
        AggregatorValidatorInterface av = validator; // cache storage reads
        if (address(av) == address(0)) return;

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

    function payOracle(uint32 _roundId) private {
        uint128 payment = details[_roundId].paymentAmount;
        Funds memory funds = recordedFunds;
        funds.available -= payment;
        funds.allocated += payment;
        recordedFunds = funds;
        oracles[msg.sender].withdrawable += payment;

        emit AvailableFundsUpdated(funds.available);
    }

    function recordSubmission(int256 _submission, uint32 _roundId) private {
        require(acceptingSubmissions(_roundId), "round not accepting submissions");

        details[_roundId].submissions.push(_submission);
        oracles[msg.sender].lastReportedRound = _roundId;
        oracles[msg.sender].latestSubmission = _submission;

        emit SubmissionReceived(_submission, _roundId, msg.sender);
    }

    function deleteRoundDetails(uint32 _roundId) private {
        if (details[_roundId].submissions.length < details[_roundId].maxSubmissions) return;

        delete details[_roundId];
    }

    function timedOut(uint32 _roundId) private view returns (bool) {
        uint64 startedAt = rounds[_roundId].startedAt;
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

    function previousAndCurrentUnanswered(
        uint32 _roundId,
        uint32 _rrId
    ) private view returns (bool) {
        return _roundId + 1 == _rrId && rounds[_rrId].updatedAt == 0;
    }

    function requiredReserve(uint256 payment) private view returns (uint256) {
        return payment * oracleCount() * RESERVE_ROUNDS;
    }

    function addOracle(address _oracle, address _admin) private {
        require(!oracleEnabled(_oracle), "oracle already enabled");

        require(_admin != address(0), "cannot set admin to 0");
        require(
            oracles[_oracle].admin == address(0) || oracles[_oracle].admin == _admin,
            "owner cannot overwrite admin"
        );

        oracles[_oracle].startingRound = getStartingRound(_oracle);
        oracles[_oracle].endingRound = ROUND_MAX;
        oracles[_oracle].index = uint16(oracleAddresses.length);
        oracleAddresses.push(_oracle);
        oracles[_oracle].admin = _admin;

        emit OraclePermissionsUpdated(_oracle, true);
        emit OracleAdminUpdated(_oracle, _admin);
    }

    function removeOracle(address _oracle) private {
        require(oracleEnabled(_oracle), "oracle not enabled");

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
        // cache storage reads
        uint32 startingRound = oracles[_oracle].startingRound;
        uint32 rrId = reportingRoundId;

        if (startingRound == 0) return "not enabled oracle";
        if (startingRound > _roundId) return "not yet enabled oracle";
        if (oracles[_oracle].endingRound < _roundId) return "no longer allowed oracle";
        if (oracles[_oracle].lastReportedRound >= _roundId)
            return "cannot report on previous rounds";
        if (
            _roundId != rrId &&
            _roundId != rrId + 1 &&
            !previousAndCurrentUnanswered(_roundId, rrId)
        ) return "invalid round to report";
        if (_roundId != 1 && !supersedable(_roundId - 1)) return "previous round not supersedable";

        return "";
    }

    function supersedable(uint32 _roundId) private view returns (bool) {
        return rounds[_roundId].updatedAt > 0 || timedOut(_roundId);
    }

    function oracleEnabled(address _oracle) private view returns (bool) {
        return oracles[_oracle].endingRound == ROUND_MAX;
    }

    function acceptingSubmissions(uint32 _roundId) private view returns (bool) {
        return details[_roundId].maxSubmissions != 0;
    }

    function delayed(address _oracle, uint32 _roundId) private view returns (bool) {
        uint256 lastStarted = oracles[_oracle].lastStartedRound;
        return _roundId > lastStarted + restartDelay || lastStarted == 0;
    }

    function newRound(uint32 _roundId) private view returns (bool) {
        return _roundId == reportingRoundId + 1;
    }

    function validRoundId(uint256 _roundId) private pure returns (bool) {
        return _roundId <= ROUND_MAX;
    }
}
