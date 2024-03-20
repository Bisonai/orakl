// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {IFeedProxy} from "./interfaces/IFeedProxy.sol";
import {ITypeAndVersion} from "./interfaces/ITypeAndVersion.sol";

/**
 * @title A trusted proxy for updating where current answers are read from
 * @notice This contract provides a consistent address for the
 * CurrentAnswerInterface but delegates where it reads from to the owner, who is
 * trusted to update it.
 */
contract FeedProxy is Ownable, IFeedProxy, ITypeAndVersion {
    IFeedProxy public feed;
    IFeedProxy private proposedFeed;

    error InvalidProposedFeed();

    event FeedProposed(address indexed current, address indexed proposed);
    event FeedConfirmed(address indexed previous, address indexed latest);

    modifier hasProposal() {
        require(address(proposedFeed) != address(0), "No proposed feed present");
        _;
    }

    constructor(address _feed) Ownable(msg.sender) {
        setFeed(_feed);
    }

    function getRoundData(uint64 roundId)
        public
        view
        virtual
        override
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return feed.getRoundData(roundId);
    }

    function latestRoundData() public view virtual override returns (uint64 id, int256 answer, uint256 updatedAt) {
        return feed.latestRoundData();
    }

    function proposedGetRoundData(uint64 roundId)
        external
        view
        virtual
        override
        hasProposal
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return proposedFeed.getRoundData(roundId);
    }

    function proposedLatestRoundData()
        external
        view
        virtual
        override
        hasProposal
        returns (uint64 id, int256 answer, uint256 updatedAt)
    {
        return proposedFeed.latestRoundData();
    }

    function getFeed() external view override returns (address) {
        return address(feed);
    }

    function decimals() external view override returns (uint8) {
        return feed.decimals();
    }

    /**
     * @inheritdoc ITypeAndVersion
     */
    function typeAndVersion() external view returns (string memory) {
        return feed.typeAndVersion();
    }

    /**
     * @notice returns the description of the feed the proxy points to.
     */
    function description() external view override returns (string memory) {
        return feed.description();
    }

    /**
     * @notice returns the current proposed feed
     */
    function getProposedFeed() external view override returns (address) {
        return address(proposedFeed);
    }

    function proposeFeed(address _feed) external onlyOwner {
        if (_feed == address(0)) {
            revert InvalidProposedFeed();
        }
        proposedFeed = IFeedProxy(_feed);
        emit FeedProposed(address(feed), _feed);
    }

    function confirmFeed(address _feed) external onlyOwner {
        if (_feed != address(proposedFeed)) {
            revert InvalidProposedFeed();
        }
        address previousFeed = address(feed);
        delete proposedFeed;
        setFeed(_feed);
        emit FeedConfirmed(previousFeed, _feed);
    }

    function setFeed(address _feed) internal {
        feed = IFeedProxy(_feed);
    }
}
