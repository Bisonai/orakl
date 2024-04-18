// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeedRouter {
    /**
     * @notice Get the address of the feed proxy given a feed name.
     * @param feedName The feed name.
     * @return The address of the feed proxy.
     */
    function feedProxies(string calldata feedName) external view returns (address);

    /**
     * @notice Update the feed proxy address of given a feed name.
     * @dev This function is restricted to the owner. If null address
     * is passed as proxy address, the function will revert with
     * `InvalidProxyAddress` error. If the proxy has sucessfully been
     * updated, the `RouterProxyAddressUpdated` event will be emitted.
     * @param feedName The feed name.
     * @param proxyAddress The address of the feed proxy.
     */
    function updateProxy(string calldata feedName, address proxyAddress) external;

    /**
     * @notice Update the feed proxy addresses in bulk.
     * @dev This function is restricted to the owner. Internally, this
     * function uses `updateProxy` to update the proxy addresses.
     * @param feedNames The feed names.
     * @param proxyAddresses The addresses of the feed proxies.
     */
    function updateProxyBulk(string[] calldata feedNames, address[] calldata proxyAddresses) external;

    /**
     * @notice Get the round data given a a feedd name and round ID.
     * @param feedName The feed name.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundData(string calldata feedName, uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get the latest round data of the feed given a feed name.
     * @param feedName The feed name.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function latestRoundData(string calldata feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get the time-weighted average price (TWAP) of the feed
     * over a given interval.
     * @param feedName The feed name.
     * @param interval The time interval in seconds
     * @param latestUpdatedAtTolerance The tolerance for the latest update time
     * @param minCount The minimum number of data points
     * @return The TWAP
     */
    function twap(string calldata feedName, uint256 interval, uint256 latestUpdatedAtTolerance, int256 minCount)
        external
        view
        returns (int256);

    /**
     * @notice Get round data from the proposed feed given a feed name and round ID.
     * @param feedName The feed name.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function proposedGetRoundData(string calldata feedName, uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get the latest round data from the proposed feed given a feed name.
     * @param feedName The feed name.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function proposedLatestRoundData(string calldata feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get address of the feed given a feed name.
     * @param feedName The feed name.
     * @return The address of the feed.
     */
    function feed(string calldata feedName) external view returns (address);

    /**
     * @notice Return the address of the proposed feed given a feed name.
     * @param feedName The feed name.
     * @return The address of the proposed feed.
     */
    function proposedFeed(string calldata feedName) external view returns (address);

    /**
     * @notice Get decimals of the feed given a feed name.
     * @param feedName The feed name.
     * @return decimals The decimals of the feed.
     */
    function decimals(string calldata feedName) external view returns (uint8);

    /**
     * @notice Get version and type of the feed given a feed name.
     * @param feedName The feed name.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion(string calldata feedName) external view returns (string memory);

    /**
     * @notice Get description of the feed given a feed name.
     * @param feedName The feed name.
     * @return description The description of the feed.
     */
    function description(string calldata feedName) external view returns (string memory);
}
