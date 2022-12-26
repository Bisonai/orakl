// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "../interfaces/AggregatorInterface.sol";

contract PriceConsumer {
    AggregatorInterface internal priceFeed;

    constructor() {
        priceFeed = AggregatorInterface(
            0xD4a33860578De61DBAbDc8BFdb98FD742fA7028e // FIXME
        );
    }

    function getLatestPrice() public view returns (int) {
       (
           /*uint80 roundID*/,
           int price,
           /*uint startedAt*/,
           /*uint timeStamp*/,
           /*uint80 answeredInRound*/
       ) = priceFeed.latestRoundData();
       return price;
    }
}
