// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IL2Aggregator {
    function submit(uint256 _roundId, int256 _submission) external;

    function latestRoundData()
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );
}
