// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IFeedRouter {
    /**
     * @notice Get the address of the feed proxy given a feed name.
     * @param feedName The feed name.
     * @return The address of the feed proxy.
     */
    function feedToProxies(string calldata feedName) external view returns (address);

    /**
     * @notice Update the feed proxy addresses in bulk.
     * @dev This function is restricted to the owner. Internally, this
     * function uses `updateProxy` to update the proxy addresses.
     * @param feedNames The feed names.
     * @param proxyAddresses The addresses of the feed proxies.
     */
    function updateProxyBulk(string[] calldata feedNames, address[] calldata proxyAddresses) external;

    /**
     * @notice Remove the feed proxy addresses in bulk.
     * @dev This function is restricted to the owner. Internally, this
     * function uses `removeProxy` to remove the proxy addresses.
     * @param feedNames The feed names.
     */
    function removeProxyBulk(string[] calldata feedNames) external;

    /**
     * @notice Get the round data given a feed name and round ID.
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
     * @notice Get the time-weighted average price (TWAP) of the
     * proposed feed over a given interval.
     * @param feedName The feed name.
     * @param interval The time interval in seconds
     * @param latestUpdatedAtTolerance The tolerance for the latest update time
     * @param minCount The minimum number of data points
     * @return The TWAP
     */
    function twapFromProposedFeed(
        string calldata feedName,
        uint256 interval,
        uint256 latestUpdatedAtTolerance,
        int256 minCount
    ) external view returns (int256);

    /**
     * @notice Get round data from the proposed feed given a feed name and round ID.
     * @param feedName The feed name.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundDataFromProposedFeed(string calldata feedName, uint64 roundId)
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
    function latestRoundDataFromProposedFeed(string calldata feedName)
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
     * @notice Get name of the feed given a feed name.
     * @param feedName The feed name.
     * @return name The name of the feed.
     */
    function name(string calldata feedName) external view returns (string memory);

    /**
     * @notice Get supported feed names.
     * @return The feed names.
     */
    function getFeedNames() external view returns (string[] memory);
}
