// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@openzeppelin/contracts/access/Ownable.sol";
import "./interfaces/IAggregatorRouter.sol";
import "./interfaces/IAggregatorProxy.sol";

/**
 * @title Orakl Network Aggregator Router
 * @notice The `AggregatorRouter` is the main contract needed to read Orakl
 * Network Data Feeds. The interface is similar to the `AggregatorProxy`
 * contract but requires an extra string parameter called `feedName`. The
 * supported `feedName` parameters are a combination of base and quote
 * currencies (e.g. BTC-USDT for Bitcoin's price in USDT stablecoin). You
 * can find all supported tokens at https://config.orakl.network.
 */
contract AggregatorRouter is Ownable, IAggregatorRouter {
    mapping(string => address) public aggregatorProxies;

    event RouterProxyAddressUpdated(string feedName, address indexed proxyAddress);
    event RouterProxyAddressBulkUpdated(string[] feedNames, address[] proxyAddresses);

    modifier validFeed(string calldata feedName) {
        require(aggregatorProxies[feedName] != address(0), "feed not set in router");
        _;
    }

    function updateProxy(string calldata feedName, address proxyAddress) external onlyOwner {
        aggregatorProxies[feedName] = proxyAddress;
        emit RouterProxyAddressUpdated(feedName, proxyAddress);
    }

    function updateProxyBulk(
        string[] calldata feedNames,
        address[] calldata proxyAddresses
    ) external onlyOwner {
        require(feedNames.length > 0 && feedNames.length == proxyAddresses.length, "invalid input");

        for (uint i = 0; i < feedNames.length; i++) {
            aggregatorProxies[feedNames[i]] = proxyAddresses[i];
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
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @dev Note that answer and updatedAt may change between queries.
     */
    function getRoundData(
        string calldata feedName,
        uint80 roundId
    )
        external
        view
        validFeed(feedName)
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return IAggregatorProxy(aggregatorProxies[feedName]).getRoundData(roundId);
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
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @dev Note that answer and updatedAt may change between queries.
     */
    function latestRoundData(
        string calldata feedName
    )
        external
        view
        validFeed(feedName)
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return IAggregatorProxy(aggregatorProxies[feedName]).latestRoundData();
    }

    /**
     * @notice Used if an aggregator contract has been proposed.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @param roundId the round ID to retrieve the round data for
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     */
    function proposedGetRoundData(
        string calldata feedName,
        uint80 roundId
    )
        external
        view
        validFeed(feedName)
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return IAggregatorProxy(aggregatorProxies[feedName]).proposedGetRoundData(roundId);
    }

    /**
     * @notice Used if an aggregator contract has been proposed.
     * @param feedName the name of the datafeed (ex. BTC-USDT)
     * @return id is the round ID for which data was retrieved
     * @return answer is the answer for the given round
     * @return startedAt is the timestamp when the round was started.
     * (Only some AggregatorV3Interface implementations return meaningful values)
     * @return updatedAt is the timestamp when the round last was updated (i.e.
     * answer was last computed)
     * @return answeredInRound is the round ID of the round in which the answer
     * was computed.
     */
    function proposedLatestRoundData(
        string calldata feedName
    )
        external
        view
        validFeed(feedName)
        returns (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        )
    {
        return IAggregatorProxy(aggregatorProxies[feedName]).proposedLatestRoundData();
    }

    /**
     * @notice returns the current phase's aggregator address.
     */
    function aggregator(
        string calldata feedName
    ) external view validFeed(feedName) returns (address) {
        return IAggregatorProxy(aggregatorProxies[feedName]).aggregator();
    }

    /**
     * @notice returns the current phase's ID.
     */
    function phaseId(string calldata feedName) external view validFeed(feedName) returns (uint16) {
        return IAggregatorProxy(aggregatorProxies[feedName]).phaseId();
    }

    /**
     * @notice represents the number of decimals the aggregator responses represent.
     */
    function decimals(string calldata feedName) external view validFeed(feedName) returns (uint8) {
        return IAggregatorProxy(aggregatorProxies[feedName]).decimals();
    }

    /**
     * @notice the type and version of aggregator to which proxy
     * points to.
     */
    function typeAndVersion(
        string calldata feedName
    ) external view validFeed(feedName) returns (string memory) {
        return IAggregatorProxy(aggregatorProxies[feedName]).typeAndVersion();
    }

    /**
     * @notice returns the description of the aggregator the proxy points to.
     */
    function description(
        string calldata feedName
    ) external view validFeed(feedName) returns (string memory) {
        return IAggregatorProxy(aggregatorProxies[feedName]).description();
    }

    /**
     * @notice returns the current proposed aggregator
     */
    function proposedAggregator(
        string calldata feedName
    ) external view validFeed(feedName) returns (address) {
        return IAggregatorProxy(aggregatorProxies[feedName]).proposedAggregator();
    }

    /**
     * @notice return a phase aggregator using the phaseId
     *
     * @param phaseId_ uint16
     */
    function phaseAggregators(
        string calldata feedName,
        uint16 phaseId_
    ) external view validFeed(feedName) returns (address) {
        return IAggregatorProxy(aggregatorProxies[feedName]).phaseAggregators(phaseId_);
    }
}
