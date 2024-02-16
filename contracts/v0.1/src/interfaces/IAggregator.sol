// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.8/interfaces/AggregatorV3Interface.sol
// https://github.com/smartcontractkit/chainlink/blob/develop/contracts/src/v0.6/interfaces/AggregatorInterface.sol

interface IAggregator {
    function decimals() external view returns (uint8);

    function description() external view returns (string memory);

    function getRoundData(
        uint80 _roundId
    )
        external
        view
        returns (
            uint80 roundId,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        );

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

    event AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt);

    event NewRound(uint256 indexed roundId, address indexed startedBy, uint256 startedAt);
}
