// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.7/dev/AggregatorProxy.sol

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IAggregatorProxy.sol";

/**
 * @title A trusted proxy for updating where current answers are read from
 * @notice This contract provides a consistent address for the
 * CurrentAnswerInterface but delegates where it reads from to the owner, who is
 * trusted to update it.
 */
contract AggregatorProxy is Ownable, IAggregatorProxy {
    struct Phase {
        uint16 id;
        IAggregatorProxy aggregator;
    }
    IAggregatorProxy private sProposedAggregator;
    mapping(uint16 => IAggregatorProxy) private sPhaseAggregators;
    Phase private sCurrentPhase;

    uint256 private constant PHASE_OFFSET = 64;
    uint256 private constant PHASE_SIZE = 16;
    uint256 private constant MAX_ID = 2 ** (PHASE_OFFSET + PHASE_SIZE) - 1;

    error InvalidProposedAggregator();

    event AggregatorProposed(address indexed current, address indexed proposed);
    event AggregatorConfirmed(address indexed previous, address indexed latest);

    modifier hasProposal() {
        require(address(sProposedAggregator) != address(0), "No proposed aggregator present");
        _;
    }

    constructor(address aggregatorAddress) {
        setAggregator(aggregatorAddress);
    }

    /**
     * @notice get data about a round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * Note that different underlying implementations of AggregatorV3Interface
     * have slightly different semantics for some of the return values. Consumers
     * should determine what implementations they expect to receive
     * data from and validate that they can properly handle return data from all
     * of them.
     * @param roundId the requested round ID as presented through the proxy, this
     * is made up of the aggregator's round ID with the phase ID encoded in the
     * two highest order bytes
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @dev Note that answer and updatedAt may change between queries.
     */
    function getRoundData(
        uint80 roundId
    )
        public
        view
        virtual
        override
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        (uint16 _phaseId, uint64 aggregatorRoundId) = parseIds(roundId);

        (id, answer, startedAt, updatedAt, answeredInRound) = sPhaseAggregators[_phaseId]
            .getRoundData(aggregatorRoundId);

        return addPhaseIds(id, answer, startedAt, updatedAt, answeredInRound, _phaseId);
    }

    /**
     * @notice get data about the latest round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * Note that different underlying implementations of AggregatorV3Interface
     * have slightly different semantics for some of the return values. Consumers
     * should determine what implementations they expect to receive
     * data from and validate that they can properly handle return data from all
     * of them.
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @dev Note that answer and updatedAt may change between queries.
     */
    function latestRoundData()
        public
        view
        virtual
        override
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        Phase memory current = sCurrentPhase; // cache storage reads

        (id, answer, startedAt, updatedAt, answeredInRound) = current.aggregator.latestRoundData();

        return addPhaseIds(id, answer, startedAt, updatedAt, answeredInRound, current.id);
    }

    /**
     * @notice Used if an aggregator contract has been proposed.
     * @param roundId the round ID to retrieve the round data for
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     */
    function proposedGetRoundData(
        uint80 roundId
    )
        external
        view
        virtual
        override
        hasProposal
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return sProposedAggregator.getRoundData(roundId);
    }

    /**
     * @notice Used if an aggregator contract has been proposed.
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     */
    function proposedLatestRoundData()
        external
        view
        virtual
        override
        hasProposal
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return sProposedAggregator.latestRoundData();
    }

    /**
     * @notice returns the current phase's aggregator address.
     */
    function aggregator() external view override returns (address) {
        return address(sCurrentPhase.aggregator);
    }

    /**
     * @notice returns the current phase's ID.
     */
    function phaseId() external view override returns (uint16) {
        return sCurrentPhase.id;
    }

    /**
     * @notice represents the number of decimals the aggregator responses represent.
     */
    function decimals() external view override returns (uint8) {
        return sCurrentPhase.aggregator.decimals();
    }

    /**
     * @inheritdoc IAggregatorProxy
     */
    function typeAndVersion() external view returns (string memory) {
        return sCurrentPhase.aggregator.typeAndVersion();
    }

    /**
     * @notice returns the description of the aggregator the proxy points to.
     */
    function description() external view override returns (string memory) {
        return sCurrentPhase.aggregator.description();
    }

    /**
     * @notice returns the current proposed aggregator
     */
    function proposedAggregator() external view override returns (address) {
        return address(sProposedAggregator);
    }

    /**
     * @notice return a phase aggregator using the phaseId
     *
     * @param phaseId_ uint16
     */
    function phaseAggregators(uint16 phaseId_) external view override returns (address) {
        return address(sPhaseAggregators[phaseId_]);
    }

    /**
     * @notice Allows the owner to propose a new address for the aggregator
     * @param aggregatorAddress The new address for the aggregator contract
     */
    function proposeAggregator(address aggregatorAddress) external onlyOwner {
        sProposedAggregator = IAggregatorProxy(aggregatorAddress);
        emit AggregatorProposed(address(sCurrentPhase.aggregator), aggregatorAddress);
    }

    /**
     * @notice Allows the owner to confirm and change the address
     * to the proposed aggregator
     * @dev Reverts if the given address doesn't match what was previously
     * proposed
     * @param aggregatorAddress The new address for the aggregator contract
     */
    function confirmAggregator(address aggregatorAddress) external onlyOwner {
        if (aggregatorAddress != address(sProposedAggregator)) {
            revert InvalidProposedAggregator();
        }
        address previousAggregator = address(sCurrentPhase.aggregator);
        delete sProposedAggregator;
        setAggregator(aggregatorAddress);
        emit AggregatorConfirmed(previousAggregator, aggregatorAddress);
    }

    function setAggregator(address aggregatorAddress) internal {
        uint16 id = sCurrentPhase.id + 1;
        sCurrentPhase = Phase(id, IAggregatorProxy(aggregatorAddress));
        sPhaseAggregators[id] = IAggregatorProxy(aggregatorAddress);
    }

    function addPhase(uint16 phase, uint64 originalId) internal pure returns (uint80) {
        return uint80((uint256(phase) << PHASE_OFFSET) | originalId);
    }

    function parseIds(uint256 roundId) internal pure returns (uint16, uint64) {
        uint16 _phaseId = uint16(roundId >> PHASE_OFFSET);
        uint64 aggregatorRoundId = uint64(roundId);

        return (_phaseId, aggregatorRoundId);
    }

    function addPhaseIds(
        uint80 roundId,
        int256 answer,
        uint256 startedAt,
        uint256 updatedAt,
        uint80 answeredInRound,
        uint16 _phaseId
    ) internal pure returns (uint80, int256, uint256, uint256, uint80) {
        return (
            addPhase(_phaseId, uint64(roundId)),
            answer,
            startedAt,
            updatedAt,
            addPhase(_phaseId, uint64(answeredInRound))
        );
    }
}
