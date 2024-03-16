// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import {IFeed} from "./IFeed.sol";

interface IFeedProxy is IFeed {
    function getProposedFeed() external view returns (address);

    function proposedGetRoundData(uint64 roundId)
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function proposedLatestRoundData()
        external
        view
        returns (uint64 id, int256 answer, uint256 updatedAt);

    function getFeed() external view returns (address);

    /**
     * @notice the type and version of aggregator to which proxy
     * points to.
     */
    function typeAndVersion() external view returns (string memory);
}
