// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

interface IFeed {
    /**
     * @notice Return the decimals of the feed.
     * @return decimals The decimals of the feed.
     */
    function decimals() external view returns (uint8);

    /**
     * @notice Return the description of the feed.
     * @return description The description of the feed.
     */
    function description() external view returns (string memory);

    /**
     * @notice Return the round data given a round ID.
     * @param _roundId The round ID.
     * @return roundId The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function getRoundData(uint64 _roundId)
        external
        view
        returns (uint64 roundId, int256 answer, uint256 updatedAt);

    /**
     * @notice Return the latest round data of the feed.
     * @dev This function internally calls getRoundData with the
     * latest round ID.
     * @return roundId The round ID.
     * @return answer The oracle answer.
     * @return updatedAt Timestamp of the last update.
     */
    function latestRoundData()
        external
        view
        returns (uint64 roundId, int256 answer, uint256 updatedAt);
}
