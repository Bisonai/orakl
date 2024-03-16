// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeedRouter {
    function feedProxies(string calldata feedName) external view returns (address);

    function updateProxy(string calldata feedName, address proxyAddress) external;

    function updateProxyBulk(string[] calldata feedNames, address[] calldata proxyAddresses) external;

    function getRoundData(string calldata feedName, uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function latestRoundData(string calldata feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function proposedGetRoundData(string calldata feedName, uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function proposedLatestRoundData(string calldata feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function feed(string calldata feedName) external view returns (address);

    function decimals(string calldata feedName) external view returns (uint8);

    function typeAndVersion(string calldata feedName) external view returns (string memory);

    function description(string calldata feedName) external view returns (string memory);

    function proposedFeed(string calldata feedName) external view returns (address);
}
