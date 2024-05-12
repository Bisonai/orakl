// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeedRouter} from "./interfaces/IFeedRouter.sol";
import {IFeedProxy} from "./interfaces/IFeedProxy.sol";

/**
 * @title Orakl Network Feed Router
 * @author Bisonai
 * @notice The `FeedRouter` is the main contract needed to read Orakl
 * Network Data Feeds. The interface is similar to the `FeedProxy`
 * contract but requires an extra feed name (`_feedName`) parameter. The
 * supported feed names are a combination of base and quote
 * symbols (e.g. BTC-USDT for Bitcoin's price in USDT stablecoin). You
 * can find all supported tokens at https://config.orakl.network.
 */
contract FeedRouter is Ownable, IFeedRouter {
    mapping(string => address) public feedToProxies;
    string[] public feedNames;

    event ProxyAdded(string feedName, address indexed proxyAddress);
    event ProxyRemoved(string feedName, address indexed proxyAddress);
    event ProxyUpdated(string feedName, address indexed proxyAddress);

    error InvalidProxyAddress();
    error InvalidInput();
    error FeedNotSetInRouter();

    modifier validFeed(string calldata _feedName) {
        if (feedToProxies[_feedName] == address(0)) {
            revert FeedNotSetInRouter();
        }
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
        if (_feedNames.length == 0 || _feedNames.length != _proxyAddresses.length) {
            revert InvalidInput();
        }

        for (uint256 i = 0; i < _feedNames.length; i++) {
            updateProxy(_feedNames[i], _proxyAddresses[i]);
        }
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function removeProxyBulk(string[] calldata _feedNames) external onlyOwner {
        if (_feedNames.length == 0) {
            revert InvalidInput();
        }

        for (uint256 i = 0; i < _feedNames.length; i++) {
            removeProxy(_feedNames[i], feedToProxies[_feedNames[i]]);
        }
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function getRoundData(string calldata _feedName, uint64 _roundId)
        external
        view
        validFeed(_feedName)
        returns (uint64, int256, uint256)
    {
        return IFeedProxy(feedToProxies[_feedName]).getRoundData(_roundId);
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function latestRoundData(string calldata _feedName)
        external
        view
        validFeed(_feedName)
        returns (uint64, int256, uint256)
    {
        return IFeedProxy(feedToProxies[_feedName]).latestRoundData();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function twap(string calldata _feedName, uint256 _interval, uint256 _latestUpdatedAtTolerance, int256 _minCount)
        external
        view
        validFeed(_feedName)
        returns (int256)
    {
        return IFeedProxy(feedToProxies[_feedName]).twap(_interval, _latestUpdatedAtTolerance, _minCount);
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function twapFromProposedFeed(
        string calldata _feedName,
        uint256 _interval,
        uint256 _latestUpdatedAtTolerance,
        int256 _minCount
    ) external view validFeed(_feedName) returns (int256) {
        return
            IFeedProxy(feedToProxies[_feedName]).twapFromProposedFeed(_interval, _latestUpdatedAtTolerance, _minCount);
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function getRoundDataFromProposedFeed(string calldata _feedName, uint64 _roundId)
        external
        view
        validFeed(_feedName)
        returns (uint64, int256, uint256)
    {
        return IFeedProxy(feedToProxies[_feedName]).getRoundDataFromProposedFeed(_roundId);
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function latestRoundDataFromProposedFeed(string calldata _feedName)
        external
        view
        validFeed(_feedName)
        returns (uint64, int256, uint256)
    {
        return IFeedProxy(feedToProxies[_feedName]).latestRoundDataFromProposedFeed();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function feed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedToProxies[_feedName]).getFeed();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function proposedFeed(string calldata _feedName) external view validFeed(_feedName) returns (address) {
        return IFeedProxy(feedToProxies[_feedName]).getProposedFeed();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function decimals(string calldata _feedName) external view validFeed(_feedName) returns (uint8) {
        return IFeedProxy(feedToProxies[_feedName]).decimals();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function typeAndVersion(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedToProxies[_feedName]).typeAndVersion();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function name(string calldata _feedName) external view validFeed(_feedName) returns (string memory) {
        return IFeedProxy(feedToProxies[_feedName]).name();
    }

    /**
     * @inheritdoc IFeedRouter
     */
    function getFeedNames() external view returns (string[] memory) {
        return feedNames;
    }

    /**
     * @notice Update the feed proxy address of given a feed name.
     * @param _feedName The feed name.
     * @param _proxyAddress The address of the feed proxy.
     */
    function updateProxy(string calldata _feedName, address _proxyAddress) private {
        if (_proxyAddress == address(0)) {
            revert InvalidProxyAddress();
        }
        bool found = false;
        if (feedToProxies[_feedName] != address(0)) {
            found = true;
        }

        feedToProxies[_feedName] = _proxyAddress;

        if (!found) {
            feedNames.push(_feedName);
            emit ProxyAdded(_feedName, _proxyAddress);
        } else {
            emit ProxyUpdated(_feedName, _proxyAddress);
        }
    }

    /**
     * @notice Remove the feed proxy address of given a feed name.
     * @param _feedName The feed name.
     * @param _proxyAddress The address of the feed proxy.
     */
    function removeProxy(string calldata _feedName, address _proxyAddress) private {
        if (_proxyAddress == address(0)) {
            revert InvalidProxyAddress();
        }

        feedToProxies[_feedName] = address(0);
        bytes32 feedNameHash = keccak256(abi.encodePacked(_feedName));

        uint256 feedNamesLength = feedNames.length;
        for (uint256 i = 0; i < feedNamesLength; i++) {
            if (keccak256(abi.encodePacked(feedNames[i])) == feedNameHash) {
                feedNames[i] = feedNames[feedNames.length - 1];
                feedNames.pop();
                break;
            }
        }

        emit ProxyRemoved(_feedName, _proxyAddress);
    }
}
