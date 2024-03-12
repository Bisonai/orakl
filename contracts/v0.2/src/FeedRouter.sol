// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeedRouter} from "./interfaces/IFeedRouter.sol";
import {IFeedProxy} from "./interfaces/IFeedProxy.sol";

/**
 * @title Orakl Network Aggregator Router
 * @notice The `FeedRouter` is the main contract needed to read Orakl
 * Network Data Feeds. The interface is similar to the `AggregatorProxy`
 * contract but requires an extra string parameter called `feedName`. The
 * supported `feedName` parameters are a combination of base and quote
 * currencies (e.g. BTC-USDT for Bitcoin's price in USDT stablecoin). You
 * can find all supported tokens at https://config.orakl.network.
 */
contract FeedRouter is Ownable, IFeedRouter {
    mapping(string => address) public feedProxies;

    event RouterProxyAddressUpdated(string feedName, address indexed proxyAddress);
    event RouterProxyAddressBulkUpdated(string[] feedNames, address[] proxyAddresses);

    modifier validFeed(string calldata feedName) {
        require(feedProxies[feedName] != address(0), "feed not set in router");
        _;
    }

    constructor() Ownable(msg.sender) {}

    function updateProxy(string calldata feedName, address proxyAddress) external onlyOwner {
        feedProxies[feedName] = proxyAddress;
        emit RouterProxyAddressUpdated(feedName, proxyAddress);
    }

    function updateProxyBulk(string[] calldata feedNames, address[] calldata proxyAddresses) external onlyOwner {
        require(feedNames.length > 0 && feedNames.length == proxyAddresses.length, "invalid input");

        for (uint256 i = 0; i < feedNames.length; i++) {
            feedProxies[feedNames[i]] = proxyAddresses[i];
        }

        emit RouterProxyAddressBulkUpdated(feedNames, proxyAddresses);
    }

    /**
     * @notice get data about a round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * Note that different underlying implementations of AggregatorV3Interface
     * have slightly different semantics for some of the return values. Consumers
     * should determine what implementations they expect to receive
     * data from and validate that they can properly handle return data from all
     * of them.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @param roundId the requested round ID as presented through the proxy, this
     * is made up of the aggregator's round ID with the phase ID encoded in the
     * two highest order bytes
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function getRoundData(string calldata feedName, uint80 roundId)
        external
        view
        validFeed(feedName)
        returns (uint80 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[feedName]).getRoundData(roundId);
    }

    /**
     * @notice get data about the latest round. Consumers are encouraged to check
     * that they're receiving fresh data by inspecting the updatedAt and
     * answeredInRound return values.
     * Note that different underlying implementations of AggregatorV3Interface
     * have slightly different semantics for some of the return values. Consumers
     * should determine what implementations they expect to receive
     * data from and validate that they can properly handle return data from all
     * of them.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function latestRoundData(string calldata feedName)
        external
        view
        validFeed(feedName)
        returns (uint80 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[feedName]).latestRoundData();
    }

    /**
     * @notice Used if an feed contract has been proposed.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @param roundId the round ID to retrieve the round data for
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function proposedGetRoundData(string calldata feedName, uint80 roundId)
        external
        view
        validFeed(feedName)
        returns (uint80 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[feedName]).proposedGetRoundData(roundId);
    }

    /**
     * @notice Used if an feed contract has been proposed.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function proposedLatestRoundData(string calldata feedName)
        external
        view
        validFeed(feedName)
        returns (uint80 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[feedName]).proposedLatestRoundData();
    }

    /**
     * @notice returns the current phase's feed address.
     */
    function feed(string calldata feedName) external view validFeed(feedName) returns (address) {
        return IFeedProxy(feedProxies[feedName]).getFeed();
    }

    /**
     * @notice represents the number of decimals the feed responses represent.
     */
    function decimals(string calldata feedName) external view validFeed(feedName) returns (uint8) {
        return IFeedProxy(feedProxies[feedName]).decimals();
    }

    /**
     * @notice the type and version of feed to which proxy
     * points to.
     */
    function typeAndVersion(string calldata feedName) external view validFeed(feedName) returns (string memory) {
        return IFeedProxy(feedProxies[feedName]).typeAndVersion();
    }

    /**
     * @notice returns the description of the feed the proxy points to.
     */
    function description(string calldata feedName) external view validFeed(feedName) returns (string memory) {
        return IFeedProxy(feedProxies[feedName]).description();
    }

    /**
     * @notice returns the current proposed feed
     */
    function proposedFeed(string calldata feedName) external view validFeed(feedName) returns (address) {
        return IFeedProxy(feedProxies[feedName]).getProposedFeed();
    }
}
