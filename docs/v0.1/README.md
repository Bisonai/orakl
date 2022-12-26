# v0.1

The oracle version v0.1 uses Solidity `^0.8.16`.

## Installation

```
yarn install @bisonai-cic/icn-contracts@v0.1
```

## Products

* [Data Feed](#data-feed)
* [Request-Response](#request-response)
* [Verifiable Random Function](#verifiable-random-function)

### Data Feed

**Data Feed** provides the latest off-chain information free of charge.
The list of data feeds can be found at [Data Feeds page](data-feeds.md).

#### Example

```solidity
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

### Request-Response

**Request-Response** allows to query any information off-chain and bring them to your smart contract.
The most common and complicated requests are predefined in the off-chain oracle, therefore they simplify on-chain request and postprocessing as well.

#### Predefined Data Feed

The list of predefined data feeds can be found at [Predefined Data Feeds page](predefined-data-feeds.md).

#### Any API

##### HTTP GET Single Word Response

```solidity
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
 * HTTP GET Multi-Variable Word Responses
 * HTTP GET Element in Array Response
 * HTTP GET Large Responses
 * Existing Job Request
-->

### Verifiable Random Function

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import "@bisonai-cic/icn-contracts/src/0.1/VRFConsumerBase.sol";
import "@bisonai-cic/icn-contracts/src/0.1/ConfirmedOwner.sol";
import "@bisonai-cic/icn-contracts/src/0.1/interfaces/VRFCoordinator.sol";

contract VRFConsumer is VRFConsumerBase {
  uint256 public s_randomResult;
  address private s_owner;

  VRFCoordinatorInterface COORDINATOR;

  error OnlyOwner(address notOwner);

  modifier onlyOwner() {
      if (msg.sender != s_owner) {
          revert OnlyOwner(msg.sender);
      }
      _;
  }

  constructor(address coordinator)
      VRFConsumerBase(coordinator)
      ConfirmedOwner(msg.sender)
  {
      s_owner = msg.sender;
      COORDINATOR = VRFCoordinatorInterface(coordinator);
  }

  function requestRandomWords() public returns(uint256 requestId) {
    bytes32 keyHash = 0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024;
    uint64 subId = 1;
    uint16 requestConfirmations = 3;
    uint32 callbackGasLimit = 1_000_000;
    uint32 numWords = 1;

    requestId = COORDINATOR.requestRandomWords(
      keyHash,
      subId,
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

  function fulfillRandomWords(uint256 /* requestId */, uint256[] memory randomWords) internal override {
    // requestId should be checked if it matches the expected request
    s_randomResult = (randomWords[0] % 50) + 1;
  }
}

```

## Payments

* Prepayment
* Subscription
* Direct (Pay as you go?)
