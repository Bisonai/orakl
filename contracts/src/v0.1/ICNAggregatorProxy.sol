// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "./interfaces/IAggregatorProxy.sol";

/**
 * @title Aggregator proxy for updating where answers can be read from
 */

contract AggregatorProxy is IAggregatorProxy {
    /**
     * @notice keeps track of different aggregators
     */
    struct Phase {
        uint16 id;
        AggregatorProxy aggregator;
    }

    Phase private currentPhase;
    AggregatorProxy public proposedAggregator;
    mapping(uint16 => AggregatorProxy) public phaseAggregators;

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
        currentPhase = Phase(id, AggregatorProxy(_aggregator));
        phaseAggregators[id] = AggregatorProxy(_aggregator);
    }

    /**
     * @notice Reads the current answer from aggregator delegated to
     */
    function latestAnswer() public view virtual returns (int256 answer) {
        return currentPhase.aggregator.latestAnswer();
    }
}
