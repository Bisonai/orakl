// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeedRouter} from "./interfaces/IFeedRouter.sol";
import {IFeedProxy} from "./interfaces/IFeedProxy.sol";

/**
 * @title Orakl Network Feed Router
 * @author Bisonai Labs
 * @notice The `FeedRouter` is the main contract needed to read Orakl
 * Network Data Feeds. The interface is similar to the `FeedProxy`
 * contract but requires an extra feed name (`_feedName`) parameter. The
 * supported feed names are a combination of base and quote
 * currencies (e.g. BTC-USDT for Bitcoin's price in USDT stablecoin). You
 * can find all supported tokens at https://config.orakl.network.
 */
contract FeedRouter is Ownable, IFeedRouter {
    mapping(string => address) public feedProxies;

    event RouterProxyAddressUpdated(string feedName, address indexed proxyAddress);
    event RouterProxyAddressBulkUpdated(string[] feedNames, address[] proxyAddresses);

    error InvalidProxyAddress();

    modifier validFeed(string calldata _feedName) {
        require(feedProxies[_feedName] != address(0), "Feed not set in router");
        _;
    }

    /**
     * @notice Construct a new `FeedRouter` contract.
     * @dev The deployer of the contract will become the owner.
     */
    constructor() Ownable(msg.sender) {}

    /**
     * @inheritdoc IFeedRouter
     */
    function updateProxyBulk(string[] calldata _feedNames, address[] calldata _proxyAddresses) external onlyOwner {
        require(_feedNames.length > 0 && _feedNames.length == _proxyAddresses.length, "invalid input");

        for (uint256 i = 0; i < _feedNames.length; i++) {
            updateProxy(_feedNames[i], _proxyAddresses[i]);
        }

        emit RouterProxyAddressBulkUpdated(_feedNames, _proxyAddresses);
    }

    /**
     * @inheritdoc IFeedRouter
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
     * @inheritdoc IFeedRouter
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
     * @inheritdoc IFeedRouter
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
     * @inheritdoc IFeedRouter
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
     * @inheritdoc IFeedRouter
     */
    function feed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedProxies[_feedName]).getFeed();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function proposedFeed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedProxies[_feedName]).getProposedFeed();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function decimals(string calldata _feedName) external view validFeed(_feedName) returns (uint8) {
        return IFeedProxy(feedProxies[_feedName]).decimals();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function typeAndVersion(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedProxies[_feedName]).typeAndVersion();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function description(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedProxies[_feedName]).description();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function updateProxy(string calldata _feedName, address _proxyAddress) public onlyOwner {
        if (_proxyAddress == address(0)) {
            revert InvalidProxyAddress();
        }

        feedProxies[_feedName] = _proxyAddress;
        emit RouterProxyAddressUpdated(_feedName, _proxyAddress);
    }
}
