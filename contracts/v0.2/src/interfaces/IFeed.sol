// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeed {
    function decimals() external view returns (uint8);

    function description() external view returns (string memory);

    function getRoundData(uint80 _roundId)
        external
        view
        returns (uint80 roundId, int256 answer, uint256 updatedAt);

    function latestRoundData()
        external
        view
        returns (uint80 roundId, int256 answer, uint256 updatedAt);
}
