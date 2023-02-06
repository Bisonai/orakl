# v0.1

Orakl is a decentralized oracle network that allows smart contracts to securely access off-chain data and other resources.

The oracle version `v0.1` uses Solidity version `^0.8.16`.

## Installation

```
yarn install @bisonai-cic/icn-contracts@v0.1
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

import "@bisonai-cic/icn-contracts/src/0.1/interfaces/AggregatorInterface.sol";

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
The most common and complicated job requests are predefined in the off-chain oracle, therefore they simplify on-chain request and postprocessing as well.

* [Predefined Job Request](#predefined-job-request)
* [Any API - HTTP GET Single Word Response](#any-api---http-get-single-word-response)

### Predefined Job Request

The list of predefined job requests can be found at [Predefined Job Requests page](predefined-job-requests.md).

### Any API - HTTP GET Single Word Response

To request data from any API one must build a request (`buildRequest`) specifying `jobId`, address of contract to fulfill and its function selector.
Request is build through `add` methods on `ICN.request` object that accept key-value pairs in a form of strings.

* `get`
* `path`

The function used for fulfillment must have parameters; `_requestId` defined as `bytes32` and fulfilling value `_response`.
`_response` type can be one of the types shown in the table below.
The response type is requested through `jobId`.

| Response type | Job ID                                          |
|---------------|-------------------------------------------------|
| `int256`      | `keccak256(abi.encodePacked("any-api-int256"))` |
| `int128`      | `keccak256(abi.encodePacked("any-api-int128"))` |
| `int64`       | `keccak256(abi.encodePacked("any-api-int64"))`  |
| `int32`       | `keccak256(abi.encodePacked("any-api-int32"))`  |

#### Example of requesting price of KLAY/USD

```Solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@bisonai-cic/icn-contracts/src/0.1/RequestResponseConsumerBase.sol";

contract AnyApiConsumer is RequestResponseConsumerBase {
    using ICN for ICN.Request;

    bytes32 private jobId;
    int256 public value;

    constructor(address _oracleAddress) {
        setOracle(_oracleAddress);
        jobId = keccak256(abi.encodePacked("any-api-int256"));
    }

    function requestData() public returns (bytes32 requestId) {
        ICN.Request memory req = buildRequest(jobId, address(this), this.fulfill.selector);
        req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=KLAY&tsyms=USD");
        req.add("path", "RAW,KLAY,USD,PRICE");
        return sendRequest(req);
    }

    function fulfill(bytes32 _requestId, int256 _response) public ICNResponseFulfilled(_requestId) {
        value = _response;
    }
}
```

<!--
### Any API - HTTP GET Multi-Variable Word Responses
### Any API - HTTP GET Element in Array Response
### Any API - HTTP GET Large Responses
-->

## Verifiable Random Function (VRF)

The information describing how to use VRF can be found at [VRF](vrf.md).
If you want to start using VRF right away, we recommend you to look at an example Hardhat project using Orakl VRF.

## Payments

* Prepayment
* Direct
