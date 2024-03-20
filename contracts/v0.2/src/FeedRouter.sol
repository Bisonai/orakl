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

    error InvalidProxyAddress();

    event RouterProxyAddressUpdated(string feedName, address indexed proxyAddress);
    event RouterProxyAddressBulkUpdated(string[] feedNames, address[] proxyAddresses);

    modifier validFeed(string calldata _feedName) {
        require(feedProxies[_feedName] != address(0), "feed not set in router");
        _;
    }

    constructor() Ownable(msg.sender) {}

    function updateProxyBulk(string[] calldata _feedNames, address[] calldata _proxyAddresses) external onlyOwner {
        require(_feedNames.length > 0 && _feedNames.length == _proxyAddresses.length, "invalid input");

        for (uint256 i = 0; i < _feedNames.length; i++) {
            updateProxy(_feedNames[i], _proxyAddresses[i]);
        }

        emit RouterProxyAddressBulkUpdated(_feedNames, _proxyAddresses);
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
     * @param _feedName the name of the datafeed (ex. BTC-USDT)
     * @param _roundId the requested round ID as presented through the proxy, this
     * is made up of the aggregator's round ID with the phase ID encoded in the
     * two highest order bytes
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function getRoundData(string calldata _feedName, uint64 _roundId)
        external
        view
        validFeed(_feedName)
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[_feedName]).getRoundData(_roundId);
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
     * @param _feedName the name of the datafeed (ex. BTC-USDT)
     * @return id is the round ID from the aggregator for which the data was
     * retrieved combined with an phase to ensure that round IDs get larger as
     * time moves forward.
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function latestRoundData(string calldata _feedName)
        external
        view
        validFeed(_feedName)
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[_feedName]).latestRoundData();
    }

    /**
     * @notice Used if an feed contract has been proposed.
     * @param _feedName the name of the datafeed (ex. BTC-USDT)
     * @param _roundId the round ID to retrieve the round data for
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function proposedGetRoundData(string calldata _feedName, uint64 _roundId)
        external
        view
        validFeed(_feedName)
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[_feedName]).proposedGetRoundData(_roundId);
    }

    /**
     * @notice Used if an feed contract has been proposed.
     * @param _feedName the name of the datafeed (ex. BTC-USDT)
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     */
    function proposedLatestRoundData(string calldata _feedName)
        external
        view
        validFeed(_feedName)
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return IFeedProxy(feedProxies[_feedName]).proposedLatestRoundData();
    }

    /**
     * @notice returns the current phase's feed address.
     */
    function feed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedProxies[_feedName]).getFeed();
    }

    /**
     * @notice represents the number of decimals the feed responses represent.
     */
    function decimals(string calldata _feedName) external view validFeed(_feedName) returns (uint8) {
        return IFeedProxy(feedProxies[_feedName]).decimals();
    }

    /**
     * @notice the type and version of feed to which proxy
     * points to.
     */
    function typeAndVersion(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedProxies[_feedName]).typeAndVersion();
    }

    /**
     * @notice returns the description of the feed the proxy points to.
     */
    function description(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedProxies[_feedName]).description();
    }

    /**
     * @notice returns the current proposed feed
     */
    function proposedFeed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedProxies[_feedName]).getProposedFeed();
    }

    function updateProxy(string calldata _feedName, address _proxyAddress) public onlyOwner {
        if (_proxyAddress == address(0)) {
            revert InvalidProxyAddress();
        }

        feedProxies[_feedName] = _proxyAddress;
        emit RouterProxyAddressUpdated(_feedName, _proxyAddress);
    }
}
