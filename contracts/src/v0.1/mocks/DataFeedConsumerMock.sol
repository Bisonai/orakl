// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../interfaces/IAggregator.sol";

contract DataFeedConsumerMock {
    IAggregator internal priceFeed;
    int256 public sPrice;
    uint80 public sRoundID;

    constructor(address _aggregatorProxy) {
        priceFeed = IAggregator(_aggregatorProxy);
    }

    function getLatestPrice() public {
        (
            uint80 roundID,
            int256 price /*uint startedAt*/ /*uint timeStamp*/ /*uint80 answeredInRound*/,
            ,
            ,

        ) = priceFeed.latestRoundData();
        sPrice = price;
        sRoundID = roundID;
    }

    function decimals() public view returns (uint8) {
        return priceFeed.decimals();
    }
}
