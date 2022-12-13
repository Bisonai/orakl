    // SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IAggregatorProxy.sol";

/**
 * @title Aggregator proxy for updating where answers can be read from
 */

contract ICNAggregatorProxy is IAggregatorProxy {
    /**
     * @notice keeps track of different aggregators
     */
    struct Phase {
        uint16 id;
        ICNAggregatorProxy aggregator;
    }

    Phase private currentPhase;
    mapping(uint16 => ICNAggregatorProxy) public phaseAggregators;

    uint256 private constant PHASE_OFFSET = 64;
    uint256 private constant PHASE_SIZE = 16;
    uint256 private constant MAX_ID = 2 ** (PHASE_OFFSET + PHASE_SIZE) - 1;

    constructor(address _aggregator) {
        setAggregator(_aggregator);
    }

    /**
     * @notice Internal setting Aggregator
     */
    function setAggregator(address _aggregator) internal {
        uint16 id = currentPhase.id + 1;
        currentPhase = Phase(id, ICNAggregatorProxy(_aggregator));
        phaseAggregators[id] = ICNAggregatorProxy(_aggregator);
    }

    /**
     * @notice Reads the current answer from aggregator delegated to
     */
    function latestAnswer() public view virtual returns (int256 answer) {
        return currentPhase.aggregator.latestAnswer();
    }

    /**
     * @notice reads the last updated time from aggregator delegated to
     */
    function latestTimestamp() public view virtual returns (uint256 updatedAt) {
        return currentPhase.aggregator.latestTimestamp();
    }

    /**
     * @notice get past round answers
     */
    function getAnswer(uint256 _roundId) public view virtual returns (int256 answer) {
        if (_roundId > MAX_ID) return 0;

        (uint16 phaseId, uint64 aggregatorRoundid) = parseIds(_roundId);
        ICNAggregatorProxy _aggregator = phaseAggregators[phaseId];
        if (address(_aggregator) == address(0)) return 0;

        return _aggregator.getAnswer(aggregatorRoundid);
    }

    function parseIds(uint256 _roundId) internal pure returns (uint16, uint64) {
        uint16 phaseId = uint16(_roundId >> PHASE_OFFSET);
        uint64 aggregatorRoundId = uint64(_roundId);
        return (phaseId, aggregatorRoundId);
    }

    /**
     * @notice get block timestamp when an answer was last updated
     * @param _roundId the answer number to retrieve the updated timestamp for
     */
    function getTimestamp(uint256 _roundId) public view virtual returns (uint256 updatedAt) {
        if (_roundId > MAX_ID) return 0;

        (uint16 phaseId, uint64 aggregatorRoundId) = parseIds(_roundId);
        ICNAggregatorProxy _aggregator = phaseAggregators[phaseId];
        if (address(_aggregator) == address(0)) return 0;

        return _aggregator.getTimestamp(aggregatorRoundId);
    }

    /**
     * @notice get the latest completed round where the answer was updated. This
     * ID includes the proxy's phase, to make sure round IDs increase even when
     * switching to a newly deployed aggregator.
     */
    function latestRound() public view virtual returns (uint256 roundId) {
        Phase memory phase = currentPhase; // cache storage reads
        return addPhase(phase.id, uint64(phase.aggregator.latestRound()));
    }

    /**
     * @notice get data about a round.
     */
    function getRoundData(uint80 _roundId)
        public
        view
        virtual
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
    {
        (uint16 phaseId, uint64 aggregatorRoundId) = parseIds(_roundId);

        (roundId, answer, startedAt, updatedAt, answeredInRound) =
            phaseAggregators[phaseId].getRoundData(aggregatorRoundId);

        return addPhaseIds(roundId, answer, startedAt, updatedAt, answeredInRound, phaseId);
    }

    /**
     * @notice get data about the latest round.
     */
    function latestRoundData()
        public
        view
        virtual
        returns (uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound)
    {
        Phase memory current = currentPhase; // cache storage reads

        (roundId, answer, startedAt, updatedAt, answeredInRound) = current.aggregator.latestRoundData();

        return addPhaseIds(roundId, answer, startedAt, updatedAt, answeredInRound, current.id);
    }

    /**
     * @notice returns the current phase's aggregator address.
     */
    function aggregator() external view returns (address) {
        return address(currentPhase.aggregator);
    }

    /**
     * @notice returns the current phase's ID.
     */
    function getphaseId() external view returns (uint16) {
        return currentPhase.id;
    }

    function addPhaseIds(
        uint80 roundId,
        int256 answer,
        uint256 startedAt,
        uint256 updatedAt,
        uint80 answeredInRound,
        uint16 phaseId
    ) internal pure returns (uint80, int256, uint256, uint256, uint80) {
        return (
            addPhase(phaseId, uint64(roundId)), answer, startedAt, updatedAt, addPhase(phaseId, uint64(answeredInRound))
        );
    }

    function addPhase(uint16 _phase, uint64 _originalId) internal pure returns (uint80) {
        return uint80(uint256(_phase) << PHASE_OFFSET | _originalId);
    }
}
