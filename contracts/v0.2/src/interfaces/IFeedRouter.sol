// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeedRouter {
    /**
     * @notice Get the address of the feed proxy given a feed name.
     * @param _feedName The feed name.
     * @return The address of the feed proxy.
     */
    function feedProxies(string calldata _feedName) external view returns (address);

    /**
     * @notice Update the feed proxy address of given a feed name.
     * @dev This function is restricted to the owner. If null address
     * is passed as proxy address, the function will revert with
     * `InvalidProxyAddress` error. If the proxy has sucessfully been
     * updated, the `RouterProxyAddressUpdated` event will be emitted.
     * @param _feedName The feed name.
     * @param _proxyAddress The address of the feed proxy.
     */
    function updateProxy(string calldata _feedName, address _proxyAddress) external;

    /**
     * @notice Update the feed proxy addresses in bulk.
     * @dev This function is restricted to the owner. Internally, this
     * function uses `updateProxy` to update the proxy addresses.
     * @param _feedNames The feed names.
     * @param _proxyAddresses The addresses of the feed proxies.
     */
    function updateProxyBulk(string[] calldata _feedNames, address[] calldata _proxyAddresses) external;

    /**
     * @notice Get the round data given a a feedd name and round ID.
     * @param _feedName The feed name.
     * @param _roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function getRoundData(string calldata _feedName, uint64 _roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified);

    /**
     * @notice Get the latest round data of the feed given a feed name.
     * @param _feedName The feed name.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function latestRoundData(string calldata _feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified);

    /**
     * @notice Get round data from the proposed feed given a feed name and round ID.
     * @param _feedName The feed name.
     * @param _roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function proposedGetRoundData(string calldata _feedName, uint64 _roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified);

    /**
     * @notice Get the latest round data from the proposed feed given a feed name.
     * @param _feedName The feed name.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function proposedLatestRoundData(string calldata _feedName)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified);

    /**
     * @notice Get address of the feed given a feed name.
     * @param _feedName The feed name.
     * @return The address of the feed.
     */
    function feed(string calldata _feedName) external view returns (address);

    /**
     * @notice Return the address of the proposed feed given a feed name.
     * @param _feedName The feed name.
     * @return The address of the proposed feed.
     */
    function proposedFeed(string calldata _feedName) external view returns (address);

    /**
     * @notice Get decimals of the feed given a feed name.
     * @param _feedName The feed name.
     * @return decimals The decimals of the feed.
     */
    function decimals(string calldata _feedName) external view returns (uint8);

    /**
     * @notice Get version and type of the feed given a feed name.
     * @param _feedName The feed name.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion(string calldata _feedName) external view returns (string memory);

    /**
     * @notice Get description of the feed given a feed name.
     * @param _feedName The feed name.
     * @return description The description of the feed.
     */
    function description(string calldata _feedName) external view returns (string memory);
}
