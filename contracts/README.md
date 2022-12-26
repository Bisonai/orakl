# ICN contracts

## Installation

```
yarn install
```

## Compilation

```
yarn compile
```

## Run Hardhat Local Node

```
npx hardhat node
```

## Testing Oracle Requests


1. Run a local Hardhat Network

```
npx hardhat node
```

2. Run the event listening script replacing private key with PK returned from local hardhat node

```
npx hardhat run src/v0.1/scripts/eventListener.mjs --network localhost
```

## Products

* Data Feed
sss
    Aggregator.sol
    AggregatorProxy.sol

* Verifiable Random Function

    VRFConsumerBase.sol
    VRFCoordinator.sol

* Request-Response

    RequestResponseConsumerBase.sol
    RequestResponseCoordinator.sol

    * Predefined Feed
    * Any API
      * HTTP GET Single Word Response
      * HTTP GET Multi-Variable Word Responses
      * HTTP GET Element in Array Response
      * HTTP GET Large Responses
      * Existing Job Request

## Payments

* Prepayment
* Subscription
* Direct (Pay as you go?)
