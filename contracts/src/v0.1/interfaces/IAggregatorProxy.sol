// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.7/interfaces/AggregatorProxyInterface.sol

import "./IAggregator.sol";

interface IAggregatorProxy is IAggregator {
    function phaseAggregators(uint16 phaseId) external view returns (address);

    function phaseId() external view returns (uint16);

    function proposedAggregator() external view returns (address);

    function proposedGetRoundData(
        uint80 roundId
    )
        external
        view
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );

    function proposedLatestRoundData()
        external
        view
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );

    function aggregator() external view returns (address);

    /**
     * @notice the type and version of aggregator to which proxy
     * points to.
     */
    function typeAndVersion() external view returns (string memory);
}
