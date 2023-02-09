# Request-Response

A detailed example of how to use Orakl Network Request-Response can be found at example repository [`request-response-consumer`](https://github.com/Bisonai/request-response-consumer).

## What is Request-Response?

TODO general

Orakl Network Request-Response can be used with two different payment approaches:

* [Prepayment (recommended)](#prepayment-recommended)
* [Direct Payment](#direct-payment)

**Prepayment** requires user to create an account, deposit KLAY and assign consumer before being able to request for data.
It is more suitable for users that know that they will use Request-Response often and possibly from multiple smart contracts.
You can learn more about **Prepayment** at [Developer's guide for Prepayment](prepayment.md).

**Direct Payment** allows user to pay directly for Request-Response without any extra prerequisites.
This approach is a great for infrequent use, or for users that do not want to hassle with **Prepayment** settings and want to use VRF as soon as possible.

In the rest of this document, we describe both **Prepayment** and **Direct Payment** approaches that can be used for Request-Response.

## Prepayment (recommended)

We assume that at this point you have already created account through [`Prepayment` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol), deposited KLAY, and assigned consumer(s) to it.
If not, [please read how to do all the above](prepayment.md), in order to be able to continue in this guide.

After you created account (and obtained `accId`), deposited some KLAY and assigned at least one consumer, you can use it to request data and receive response.

* [Initialization](#initialization)
* [Request data](#request-data)
* [Receive response](#receive-response)

User smart contract that wants to utilize Orakl Network Request-Response has to inherit from [`VRFRequestResponseBase` abstract smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseConsumerBase.sol).

```Solidity
import "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerBase.sol";
contract RequestResponseConsumer is RequestResponseConsumerBase {
    ...
}
```

### Initialization

Request-Response smart contract ([`RequestResponseCoordinator`](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseCoordinator.sol)) is used both for requesting and receiving data.
Address of deployed `RequestResponseCoordinator` is used for initialization of parent class `RequestResponseConsumerBase` from which consumer's contract has to inherit.

```Solidity
import "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerBase.sol";

contract RequestResponseConsumer is RequestResponseConsumerBase {
  constructor(address coordinator) RequestResponseConsumerBase(coordinator) {
  }
}
```

### Request data

Request data (`requestData`) must be called from a contract that has been approved through `addConsumer` function of [`Prepayment` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol).
If the smart contract has not been approved, the request is rejected through `InvalidConsumer` error.
If account (specified by `accId`) does not exist (`InvalidAccount` error) or does not have balance high enough request is rejected as well.

```Solidity
function requestData(
  uint64 accId,
  uint32 callbackGasLimit
)
    public
    onlyOwner
    returns (uint256 requestId)
{
    bytes32 jobId = keccak256(abi.encodePacked("any-api-uint256"));
    Orakl.Request memory req = buildRequest(jobId);
    req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD");
    req.add("path", "RAW,ETH,USD,PRICE");

    requestId = COORDINATOR.requestData(
        req,
        callbackGasLimit,
        accId
    );
}
```

Below, you can find an explanation of `requestData` function and its arguments defined at [`RequestResponseCoordinator` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseCoordinator.sol):

* `req`: a `Request` structure that holds encoded user request
* `accId`: a `uint64` value representing the ID of the account associated with the request.
* `callbackGasLimit`: a `uint32` value representing the gas limit for the callback function that executes after the confirmations have been received.

The function call `requestData()` on `COORDINATOR` contract passes `req`, `accId` and `callbackGasLimit` as arguments.
After a successfull execution of this function, you obtain an ID (`requestId`) that uniquely defines your request.
Later, when your request is fulfilled, the ID (`requestId`) is supplied together with response to be able to make a match between requests and fulfillments when there is more than one request.

### Receive response

`fulfillDataRequest` is a virtual function of [`RequestResponseConsumerBase` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseConsumerBase.sol), and therefore must be overriden.
This function is called by `RequestResponseCoordinator` when fulfilling the request.
`callbackGasLimit` parameter defined during data request denotes the amount of gas required for execuction of this function.

```Solidity
function fulfillDataRequest(
    uint256 /*requestId*/,
    uint256 response
)
    internal
    override
{
    s_response = response;
}
```

The arguments of `fulfillDataRequest` function are explained below:

* `requestId`: a `uint256` value representing the ID of the request
* `response`: an `uint256` value that was obtained after processing data request sent from `requestData` function

This function is executed from `RequestResponseCoordinator` contract defined during smart contract initialization.
The result is saved in the storage variable `s_response`.

## Direct Payment

**Direct Payment** represents an alternative payment method which does not require a user to create account, deposit KLAY, and assign consumer before being able to utilize VRF functionality.
Request-Response with **Direct Payment** is only a little bit different compared to **Prepayment**, however, fulfillment function is exactly same.

* [Initialization for direct payment](#initialization-for-direct-payment)
* [Request data with direct payment (consumer)](#request-data-with-direct-payment-consumer)
* [Request data with direct payment (coordinator)](#request-data-with-direct-payment-coordinator)

User smart contract that wants to utilize Orakl Network Request-Response has to inherit from [`VRFRequestResponseBase` abstract smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/ReqeustResponseConsumerBase.sol).

```Solidity
import "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerBase.sol";
contract RequestResponseConsumer is RequestResponseConsumerBase {
    ...
}
```

### Initialization for direct payment

There is no difference in initializing Request-Response user contract that request for data with **Prepayment** or **Direct Payment**.

Request-Response smart contract ([`RequestResponseCoordinator`](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseCoordinator.sol)) is used both for requesting and receiving data.
Address of deployed `RequestResponseCoordinator` is used for initialization of parent class `RequestResponseConsumerBase` from which consumer's contract has to inherit.

```Solidity
import "@bisonai/orakl-contracts/src/v0.1/RequestResponseConsumerBase.sol";

contract RequestResponseConsumer is RequestResponseConsumerBase {
  constructor(address coordinator) RequestResponseConsumerBase(coordinator) {
  }
}
```

### Request data with direct payment (consumer)

The data request using **Direct Payment** is very similar to data request using **Prepayment**.
The only difference is that for **Direct Payment** user has to send KLAY together with call using `value` property, and does not have to specify account ID (`accId`) as in **Prepayment**.
There are several checks that has to pass in order to successully request data.
You can read about about them in one of the previous subsections called [Request data](#request-data).

```Solidity
receive() external payable {}

function requestDataDirectPayment(
  uint32 callbackGasLimit
)
    public
    payable
    onlyOwner
    returns (uint256 requestId)
{
    bytes32 jobId = keccak256(abi.encodePacked("any-api-uint256"));
    Orakl.Request memory req = buildRequest(jobId);
    req.add("get", "https://min-api.cryptocompare.com/data/pricemultifull?fsyms=ETH&tsyms=USD");
    req.add("path", "RAW,ETH,USD,PRICE");

    requestId = COORDINATOR.requestData{value: msg.value}(
        req,
        callbackGasLimit
    );
}
```

This function calls the `requestData()` function defined in `COORDINATOR` contract, and passes `req` and `callbackGasLimit` as arguments.
The payment for service is sent through `msg.value` to the `requestData()` in `COORDINATOR` contract.
If the payment is larger than expected payment, exceeding payment is returned to the caller of `requestData` function, therefore it requires the user contract to define [`receive()` function](https://docs.soliditylang.org/en/v0.8.16/contracts.html#receive-ether-function) as shown in the top of code listing.
Eventually, it generates a data request.

In the section below, you can find more detailed explanation how data request using direct payment works.

### Request data with direct payment (coordinator)

The following function is defined in [`RequestResponseCoordinator` contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/RequestResponseCoordinator.sol).

```Solidity
function requestData(
    Orakl.Request memory req,
    uint32 callbackGasLimit
) external payable returns (uint256) {
    uint256 fee = estimateDirectPaymentFee();
    if (msg.value < fee) {
        revert InsufficientPayment(msg.value, fee);
    }

    uint64 accId = s_prepayment.createAccount();
    s_prepayment.addConsumer(accId, msg.sender);
    bool isDirectPayment = true;
    uint256 requestId = requestDataInternal(req, accId, callbackGasLimit, isDirectPayment);
    s_prepayment.deposit{value: fee}(accId);

    uint256 remaining = msg.value - fee;
    if (remaining > 0) {
        (bool sent, ) = msg.sender.call{value: remaining}("");
        if (!sent) {
            revert RefundFailure();
        }
    }

    return requestId;
}
```

This function first calculates a fee (`fee`) for the request by calling `estimateDirectPaymentFee()` function.
`isDirectPayment` variable indicates whether the request is created through **Prepayment** or **Direct Payment** method.
Then, it deposits the required fee (`fee`) to the account by calling `s_prepayment.deposit(accId)` and passing the fee (`fee`) as value.
If the amount of KLAY passed by `msg.value` to the `requestData` is larger than required fee (`fee`), the remaining amount is sent back to the caller using the `msg.sender.call()` method.
Finally, the function returns `requestId` that is generated by the `requestDataInternal()` function.
