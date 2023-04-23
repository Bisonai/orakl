// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../interfaces/IAggregator.sol";

contract DataFeedConsumerMock {
    IAggregator internal priceFeed;

    uint80 public sId;
    int256 public sAnswer;
    uint256 public sStartedAt;
    uint256 public sUpdatedAt;
    uint80 public sAnsweredInRound;

    constructor(address _aggregatorProxy) {
        priceFeed = IAggregator(_aggregatorProxy);
    }

    function getLatestRoundData() public {
        (
            uint80 id,
            int256 answer,
            uint256 startedAt,
            uint256 updatedAt,
            uint80 answeredInRound
        ) = priceFeed.latestRoundData();

        sId = id;
        sAnswer = answer;
        sStartedAt = startedAt;
        sUpdatedAt = updatedAt;
        sAnsweredInRound = answeredInRound;
    }

    function decimals() public view returns (uint8) {
        return priceFeed.decimals();
    }
}
