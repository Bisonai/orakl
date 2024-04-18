// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IFeed} from "./IFeed.sol";

interface IFeedProxy is IFeed {
    /**
     * @notice Get round data from the proposed feed given a round ID.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function proposedGetRoundData(uint64 roundId) external view returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get the latest round data from the proposed feed.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function proposedLatestRoundData() external view returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get address of the feed.
     * @return The address of the feed.
     */
    function getFeed() external view returns (address);

    /**
     * @notice Get address of the proposed feed.
     * @return The address of the proposed feed.
     */
    function getProposedFeed() external view returns (address);
}
