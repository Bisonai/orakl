// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../interfaces/AggregatorInterface.sol";

contract DataFeedConsumerMock {
    AggregatorInterface internal priceFeed;
    int256 public s_price;
    uint80 public s_roundID;

    constructor(address _aggregatorProxy) {
        priceFeed = AggregatorInterface(_aggregatorProxy);
    }

    function getLatestPrice() public {
        (
            uint80 roundID,
            int256 price /*uint startedAt*/ /*uint timeStamp*/ /*uint80 answeredInRound*/,
            ,
            ,

        ) = priceFeed.latestRoundData();
        s_price = price;
        s_roundID = roundID;
    }

    function decimals() public view returns (uint8) {
        return priceFeed.decimals();
    }
}
