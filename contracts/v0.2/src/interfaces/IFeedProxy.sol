// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IFeed} from "./IFeed.sol";

interface IFeedProxy is IFeed {
    /**
     * @notice return the address of the proposed feed.
     */
    function getProposedFeed() external view returns (address);

    function proposedGetRoundData(uint64 roundId) external view returns (uint64 id, int256 answer, uint256 updatedAt);

    function proposedLatestRoundData() external view returns (uint64 id, int256 answer, uint256 updatedAt);

    function getFeed() external view returns (address);
}
