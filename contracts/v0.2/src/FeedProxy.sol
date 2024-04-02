// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeedProxy} from "./interfaces/IFeedProxy.sol";
import {IFeed} from "./interfaces/IFeed.sol";

/**
 * @title Orakl Network Proxy Feed
 * @author Bisonai Labs
 * @notice A contract that acts as a proxy for a `Feed` contract. It
 * allows the owner to propose and confirm a new feed.
 * @dev The current and proposed contracts are stored in the `feed`
 * and `proposedFeed` variables.
 */
contract FeedProxy is Ownable, IFeedProxy {
    IFeed public feed;
    IFeed private proposedFeed;

    event FeedProposed(address indexed current, address indexed proposed);
    event FeedConfirmed(address indexed previous, address indexed current);

    error InvalidProposedFeed();

    modifier hasProposal() {
        require(address(proposedFeed) != address(0), "No proposed feed present");
        _;
    }

    /**
     * @notice Construct a new `FeedProxy` contract.
     * @dev The deployer of the contract will become the owner.
     * @param _feed The address of the initial feed
     */
    constructor(address _feed) Ownable(msg.sender) {
        setFeed(_feed);
    }

    /**
     * @notice Get round data given a round ID.
     * @param _roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function getRoundData(uint64 _roundId) external view returns (uint64 id, int256 answer, uint256 updatedAt, bool verified) {
        return feed.getRoundData(_roundId);
    }

    /**
     * @notice Get timestamp of the latest round update.
     * @return The timestamp of the latest round update
     */
    function latestRoundUpdatedAt() external view returns (uint256) {
        return feed.latestRoundUpdatedAt();
    }

    /**
     * @notice Get the latest round data of the feed.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     * @return verified A boolean indicating if the data is verified.
     */
    function latestRoundData() external view returns (uint64 id, int256 answer, uint256 updatedAt, bool verified) {
        return feed.latestRoundData();
    }

    /**
     * @inheritdoc IFeedProxy
     */
    function proposedGetRoundData(uint64 _roundId)
        external
        view
        hasProposal
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified)
    {
        return proposedFeed.getRoundData(_roundId);
    }

    /**
     * @inheritdoc IFeedProxy
     */
    function proposedLatestRoundData()
        external
        view
        hasProposal
        returns (uint64 id, int256 answer, uint256 updatedAt, bool verified)
    {
        return proposedFeed.latestRoundData();
    }

    /**
     * @inheritdoc IFeedProxy
     */
    function getFeed() external view returns (address) {
        return address(feed);
    }

    /**
     * @inheritdoc IFeedProxy
     */
    function getProposedFeed() external view returns (address) {
        return address(proposedFeed);
    }

    /**
     * @notice Get decimals of the feed.
     * @return decimals The decimals of the feed.
     */
    function decimals() external view returns (uint8) {
        return feed.decimals();
    }

    /**
     * @inheritdoc IFeed
     */
    function typeAndVersion() external view returns (string memory) {
        return feed.typeAndVersion();
    }

    /**
     * @inheritdoc IFeed
     */
    function description() external view returns (string memory) {
        return feed.description();
    }

    /**
     * @notice Propose a new feed to update to. This is the first step
     * of feed update process. The second step (`confirmFeed`) is to
     * confirm the feed.
     * @dev Only the owner can propose a new feed. The feed must not
     * be the zero address. When a new feed is proposed, it emits a
     * `FeedProposed` event.
     * @param _feed The address of the new feed
     */
    function proposeFeed(address _feed) external onlyOwner {
        if (_feed == address(0)) {
            revert InvalidProposedFeed();
        }
        proposedFeed = IFeedProxy(_feed);
        emit FeedProposed(address(feed), _feed);
    }

    /**
     * @notice Confirm the proposed feed. This is the second step of
     * feed update process.
     * @dev Only the owner can confirm the feed. The proposed feed
     * must not be the zero address. When a new feed is confirmed, it
     * emits a `FeedConfirmed` event.
     * @param _feed The address of the proposed feed
     */
    function confirmFeed(address _feed) external onlyOwner {
        if (_feed != address(proposedFeed)) {
            revert InvalidProposedFeed();
        }
        address previousFeed = address(feed);
        delete proposedFeed;
        setFeed(_feed);
        emit FeedConfirmed(previousFeed, _feed);
    }

    /**
     * @notice Set the feed to a new address. This function is
     * internal and should be called by the constructor, or by the
     * `confirmFeed` function.
     * @param _feed The address of the new feed
     */
    function setFeed(address _feed) internal {
        feed = IFeedProxy(_feed);
    }
}
