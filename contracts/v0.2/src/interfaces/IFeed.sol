// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IFeed {
    /**
     * @notice Get decimals of the feed.
     * @return decimals The decimals of the feed.
     */
    function decimals() external view returns (uint8);

    /**
     * @notice Get name of the feed.
     * @return name The name of the feed.
     */
    function name() external view returns (string memory);

    /**
     * @notice Get version and type of the feed.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion() external view returns (string memory);

    /**
     * @notice Get latest round data of the feed.
     * @dev This function internally calls getRoundData with the
     * latest round ID. If no rounds have been submitted, this function
     * reverts with NoDataPresent error.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function latestRoundData() external view returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get timestamp of the latest round update
     * @dev If no updates have been made, this function returns 0.
     * @return The timestamp of the latest round update
     */
    function latestRoundUpdatedAt() external view returns (uint256);

    /**
     * @notice Get the time-weighted average price (TWAP) of the feed
     * over a given interval.
     * @dev If the latest update time is older than the given tolerance,
     * this function reverts with AnswerAboveTolerange error. If the number of
     * data points is less than the given minimum count, this function reverts
     * with InsufficientData error.
     * @param interval The time interval in seconds
     * @param latestUpdatedAtTolerance The tolerance for the latest update time
     * @param minCount The minimum number of data points
     * @return The TWAP
     */
    function twap(uint256 interval, uint256 latestUpdatedAtTolerance, int256 minCount) external view returns (int256);

    /**
     * @notice Get round data given a round ID.
     * @dev If the given roundId is higher than the latest round ID or
     * feed has not been updated yet, this function reverts with
     * NoDataPresent error.
     * @param roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundData(uint64 roundId) external view returns (uint64 id, int256 answer, uint256 updatedAt);
}
