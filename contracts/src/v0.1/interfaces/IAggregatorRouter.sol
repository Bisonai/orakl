// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

interface IAggregatorRouter {
    function aggregatorProxies(string calldata feedName) external view returns (address);

    function updateProxy(string calldata feedName, address proxyAddress) external;

    function updateProxyBulk(
        string[] calldata feedNames,
        address[] calldata proxyAddresses
    ) external;

    function getRoundData(
        string calldata feedName,
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

    function latestRoundData(
        string calldata feedName
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

    function proposedGetRoundData(
        string calldata feedName,
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

    function proposedLatestRoundData(
        string calldata feedName
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

    function aggregator(string calldata feedName) external view returns (address);

    function phaseId(string calldata feedName) external view returns (uint16);

    function decimals(string calldata feedName) external view returns (uint8);

    function typeAndVersion(string calldata feedName) external view returns (string memory);

    function description(string calldata feedName) external view returns (string memory);

    function proposedAggregator(string calldata feedName) external view returns (address);

    function phaseAggregators(
        string calldata feedName,
        uint16 phaseId_
    ) external view returns (address);
}
