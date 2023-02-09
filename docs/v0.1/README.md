# Developer's Guide (v0.1)

Orakl Network is a decentralized oracle network that allows smart contracts to securely access off-chain data and other resources.

## Installation

> The oracle version `v0.1` uses Solidity version `^0.8.16`.
> The version of npm package `@bisonai/orakl-contract` is an internal version of package to recognize changes.
> In the closed alpha version, smart contracts are all defined under v0.1 but are expected to be frequently changed.

```
yarn install @bisonai/orakl-contracts@v0.4.4
```

## Products

<!-- * [Data Feed](#data-feed) -->
* [Request-Response](#request-response)
* [Verifiable Random Function (VRF)](#verifiable-random-function-vrf)

<!--
## Data Feed

**Data Feed** provides the latest aggregated off-chain information sourced from multiple data providers.
-->

### Request-Response

**Request-Response** allows to query any information off-chain and bring it to your smart contract.

The detailed information about Orakl Request-Response can be found at [developer's guide on how to use Request-Response](request-response.md).
After understanding the basics of Request-Response, you can look at[an example Hardhat project using Orakl Request-Response](https://github.com/Bisonai/vrf-consumer).

<!--
### Request-Response - HTTP GET Single Word Response
### Request-Response - HTTP GET Multi-Variable Word Responses
### Request-Response - HTTP GET Element in Array Response
### Request-Response - HTTP GET Large Responses
-->

### Verifiable Random Function (VRF)

**Verifiable Random Function** allows for the generation of random numbers on the blockchain in a verifiable and transparent manner.
This can be useful for a wide range of use cases, such as gaming and gambling applications, where a fair and unbiased random number generator is essential.

The detailed information about Orakl VRF can be found at [developer's guide on how to use VRF](vrf.md).
If you want to start using VRF right away, we recommend you to look at [an example Hardhat project using Orakl VRF](https://github.com/Bisonai/vrf-consumer).

## Payment

Orakl Network Request-Response can be used with two different payment approaches:

* [Prepayment](#prepayment)
* [Direct Payment](#direct-payment)

### Prepayment

**Prepayment** allows consumers of Orakl Network to prepay for services, and then using those funds when interacting with Orakl Network.
It is currently accepted payment method for both VRF and Request-Response.
If you want to learn more about prepayment, go to [developer's guide on how to use Prepayment](prepayment.md).

### Direct Payment

**Direct Payment** method allows to use Orakl Network's services without requiring to create any account in advance.
All services can be requested and paid at the time of the request.
