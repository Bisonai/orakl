# Developer's Guide (v0.1)

Orakl is a decentralized oracle network that allows smart contracts to securely access off-chain data and other resources.

The oracle version `v0.1` uses Solidity version `^0.8.16`.

## Installation

```
yarn install @bisonai/orakl-contracts@v0.4.4
```

## Products

* [Data Feed](#data-feed)
* [Request-Response](#request-response)
* [Verifiable Random Function (VRF)](#verifiable-random-function-vrf)

## Data Feed

**Data Feed** provides the latest aggregated off-chain information sourced from multiple data providers.
The list of data feeds can be found at [Aggregated Data Feeds page](aggregated-data-feeds.md).

### Example

```Solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@bisonai/orakl-contracts/src/0.1/interfaces/AggregatorInterface.sol";

contract PriceConsumer {
    AggregatorInterface internal priceFeed;

    constructor(address _priceFeed) {
        priceFeed = AggregatorInterface(_priceFeed);
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

```

## Request-Response

**Request-Response** allows to query any information off-chain and bring it to your smart contract.

The detailed information about Orakl Request-Response can be found at [developer's guide on how to use Request-Response](request-response.md).
After understanding the basics of Request-Response, you can look at[an example Hardhat project using Orakl Request-Response](https://github.com/Bisonai/vrf-consumer).

<!--
### Request-Response - HTTP GET Single Word Response
### Request-Response - HTTP GET Multi-Variable Word Responses
### Request-Response - HTTP GET Element in Array Response
### Request-Response - HTTP GET Large Responses
-->

## Verifiable Random Function (VRF)

**Verifiable Random Function** allows for the generation of random numbers on the blockchain in a verifiable and transparent manner.
This can be useful for a wide range of use cases, such as gaming and gambling applications, where a fair and unbiased random number generator is essential.

The detailed information about Orakl VRF can be found at [developer's guide on how to use VRF](vrf.md).
If you want to start using VRF right away, we recommend you to look at [an example Hardhat project using Orakl VRF](https://github.com/Bisonai/vrf-consumer).

## Payment

* Prepayment
* Direct
