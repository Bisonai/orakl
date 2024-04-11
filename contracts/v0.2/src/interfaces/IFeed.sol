// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeed {
    /**
     * @notice Get decimals of the feed.
     * @return decimals The decimals of the feed.
     */
    function decimals() external view returns (uint8);

    /**
     * @notice Get description of the feed.
     * @return description The description of the feed.
     */
    function description() external view returns (string memory);

    /**
     * @notice Get round data given a round ID.
     * @param _roundId The round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundData(uint64 _roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get latest round data of the feed.
     * @dev This function internally calls getRoundData with the
     * latest round ID.
     * @return id The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function latestRoundData() external view returns (uint64 id, int256 answer, uint256 updatedAt);

    /**
     * @notice Get timestamp of the latest round update
     * @return The timestamp of the latest round update
     */
    function latestRoundUpdatedAt() external view returns (uint256);

    /**
     * @notice Get version and type of the feed.
     * @return typeAndVersion The type and version of the feed.
     */
    function typeAndVersion() external view returns (string memory);
}
