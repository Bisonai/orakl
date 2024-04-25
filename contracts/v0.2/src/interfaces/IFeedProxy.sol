// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {IFeed} from "./IFeed.sol";

interface IFeedProxy is IFeed {
    /**
     * @notice Get round data from the proposed feed given a round ID.
     * @dev If there is no proposed feed, this function reverts with
     * NoProposedFeed error. If the given roundId is higher than the
     * latest round ID or feed has not been updated yet, this function
     * reverts with NoDataPresent error.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundDataFromProposedFeed(uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get the latest round data from the proposed feed.
     * @dev If there is no proposed feed, this function reverts with
     * NoProposedFeed error. If no rounds have been submitted, this function
     * reverts with NoDataPresent error.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function latestRoundDataFromProposedFeed() external view returns (uint64 id, int256 answer, uint256 updatedAt);

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

    /**
     * @notice Get the time-weighted average price (TWAP) of the
     * proposed feed over a given interval.
     * @param interval The time interval in seconds
     * @param latestUpdatedAtTolerance The tolerance for the latest update time
     * @param minCount The minimum number of data points
     * @return The TWAP
     */
    function twapFromProposedFeed(uint256 interval, uint256 latestUpdatedAtTolerance, int256 minCount)
        external
        view
        returns (int256);
}
