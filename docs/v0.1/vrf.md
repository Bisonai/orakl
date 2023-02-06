# Verifiable Random Function (VRF)

A detailed example of how to use Orakl Network VRF can be found at example repository [`vrf-consumer`](https://github.com/Bisonai-CIC/vrf-consumer).

## What is Verifiable Random Function?

A Verifiable Random Function (VRF) is a cryptographic function that generates a random value, or output, based on some input data (called the "seed").
Importantly, the VRF output is verifiable, meaning that anyone who has access to the VRF output and the seed can verify that the output was generated correctly.

In the context of the blockchain, VRFs can be used to provide a source of randomness that is unpredictable and unbiased.
This can be useful in various decentralized applications (dApps) that require randomness as a key component, such as in randomized auctions or as part of a decentralized games.

Orakl Network VRF allows smart contracts to use VRF to generate verifiably random values, which can be used in various dApps that require randomness.
Orakl Network VRF can be used with two different payment approaches:

* [Prepayment (recommended)](#prepayment-recommended)
* [Direct Payment](#direct-payment)

**Prepayment** requires user to create an account, deposit KLAY and assign consumer before being able to request for VRF.
It is more suitable for users that know that they will use VRF often and possibly from multiple smart contracts.
You can learn more about **Prepayment** at [Developer's guide for Prepayment](prepayment.md).

**Direct Payment** allows user to pay directly for VRF without any extra prerequisites.
This approach is a great for infrequent use, or for users that do not want to hassle with **Prepayment** settings and want to use VRF as soon as possible.

In the rest of this document, we describe both **Prepayment** and **Direct Payment** approaches that can be used to request VRF.

## Prepayment (recommended)

We assume that at this point you have already created account through [`Prepayment` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol), deposited KLAY, and assigned consumer(s) to it.
If not, [please read how to do all the above](prepayment.md), in order to be able to continue in this guide.

After you created account (and obtained `accId`), deposited some KLAY and assigned at least one consumer, you can use it to request and fulfill random words.

* [Initialization](#initialization)
* [Request random words](#request-random-words)
* [Fulfill-random words](#fulfill-random-words)

User smart contract that wants to use Orakl Network VRF has to inherit from [`VRFConsumerBase` abstract smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFConsumerBase.sol).

```Solidity
import '@bisonai/orakl-contracts/src/v0.1/VRFConsumerBase.sol';
contract VRFConsumer is VRFConsumerBase {
    ...
}
```

### Initialization

VRF smart contract ([`VRFCoordinator`](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFCoordinator.sol)) is used both for requesting random words and also for request fulfillments.
We recommend you to bond `VRFCoordinator` interface with `VRFCordinator` address supplied through a constructor parameter, and use use it for random words requests (`requestRandomWords`).

```Solidity
import '@bisonai/orakl-contracts/src/v0.1/VRFConsumerBase.sol';
import '@bisonai/orakl-contracts/src/v0.1/interfaces/VRFCoordinatorInterface.sol';

contract VRFConsumer is VRFConsumerBase {
  VRFCoordinatorInterface COORDINATOR;

  constructor(address coordinator) VRFConsumerBase(coordinator) {
      COORDINATOR = VRFCoordinatorInterface(coordinator);
  }
}
```

### Request random words

Request for random words must be called from a contract that has been approved through `addConsumer` function of [`Prepayment` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol).
If the smart contract has not been approved, the request is rejected through `InvalidConsumer` error.
If account (specified by `accId`) does not exist (`InvalidAccount` error), does not have balance high enough, or uses an unregistered `keyHash` (`InvalidKeyHash` error) request is rejected as well.

```Solidity
function requestRandomWords(
    bytes32 keyHash,
    uint64 accId,
    uint32 callbackGasLimit,
    uint32 numWords
)
    public
    onlyOwner
    returns (uint256 requestId)
{
   requestId = COORDINATOR.requestRandomWords(
       keyHash,
       accId,
       callbackGasLimit,
       numWords
   );
}
```

Below, you can find an explanation of `requestRandomWords` function and its arguments defined at [`VRFCoordinator` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFCoordinator.sol):

* `keyHash`: a `bytes32` value representing the hash of the key used to generate the random words, also used to choose a trusted VRF provider.
* `accId`: a `uint64` value representing the ID of the account associated with the request.
* `callbackGasLimit`: a `uint32` value representing the gas limit for the callback function that executes after the confirmations have been received.
* `numWords`: a `uint32` value representing the number of random words requested.

The function call `requestRandomWords()` on `COORDINATOR` contract passes `keyHash`, `accId`, `callbackGasLimit`, and `numWords` as arguments.
After a successfull execution of this function, you obtain an ID (`requestId`) that uniquely defines your request.
Later, when your request is fulfilled, the ID (`requestId`) is supplied together with random words to be able to make a match between requests and fulfillments when there is more than one request.

### Fulfill random words

`fulfillRandomWords` is a virtual function of [`VRFConsumerBase` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFConsumerBase.sol), and therefore must be overriden.
This function is called by `VRFCoordinator` when fulfilling the request.
`callbackGasLimit` paramter defined during VRF request denotes the amount of gas required for execuction of this function.

```Solidity
function fulfillRandomWords(
    uint256 /* requestId */,
    uint256[] memory randomWords
)
    internal
    override
{
    // requestId should be checked if it matches the expected request
    // Generate random value between 1 and 50.
    s_randomResult = (randomWords[0] % 50) + 1;
}
```

The arguments of `fulfillRandomWords` function are explained below:

* `requestId`: `uint256` value representing the ID of the request
* `randomWords`: an array of `uint256` values representing the random words generated in response to the request

This function is executed from previously defined `COORDINATOR` contract.
After receiving random value(s) (`randomWords`) which can be any number in range of `uint256` data type, it takes the first random element and limits it to a range between 1 and 50.
The result is saved in the storage variable `s_randomResult`.


## Direct Payment

**Direct Payment** represents an alternative payment method which does not require a user to create account, deposit KLAY, and assign consumer before being able to utilize VRF functionality.
Request for VRF with **Direct Payment** is only a little bit different compared to **Prepayment**, however, fulfillment function is exactly same.

* [Initialization for direct payment](#initialization-for-direct-payment)
* [Request random words with direct payment (consumer)](#request-random-words-with-direct-payment-consumer)
* [Request random words with direct payment (coordinator)](#request-random-words-with-direct-payment-coordinator)

User smart contract that wants to use Orakl Network VRF has to inherit from [`VRFConsumerBase` abstract smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFConsumerBase.sol).

```Solidity
import '@bisonai/orakl-contracts/src/v0.1/VRFConsumerBase.sol';
contract VRFConsumer is VRFConsumerBase {
    ...
}
```

### Initialization for direct payment

There is no difference in initializing VRF user contract that request for VRF with **Prepayment** or **Direct Payment**.

VRF smart contract ([`VRFCoordinator`](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFCoordinator.sol)) is used both for requesting random words and also for request fulfillments.
We recommend you to bond `VRFCoordinator` interface with `VRFCordinator` address supplied through a constructor parameter, and use use it for random words requests (`requestRandomWordsPayment`).

```Solidity
import '@bisonai/orakl-contracts/src/v0.1/VRFConsumerBase.sol';
import '@bisonai/orakl-contracts/src/v0.1/interfaces/VRFCoordinatorInterface.sol';

contract VRFConsumer is VRFConsumerBase {
  VRFCoordinatorInterface COORDINATOR;

  constructor(address coordinator) VRFConsumerBase(coordinator) {
      COORDINATOR = VRFCoordinatorInterface(coordinator);
  }
}
```

### Request random words with direct payment (consumer)

The request for random words using **Direct Payment** is very similar to request using **Prepayment**.
The only difference is that for **Direct Payment** user has to send KLAY together with call using `value` property, and the name of function is `requestRandomWordsPayment` instead of `requestRandomWords` used for **Prepayment**.
There are several checks that has to pass in order to successully request for VRF.
You can read about about them in one of the previous subsections called [Request random words](#request-random-words).

```Solidity
receive() external payable {}

function requestRandomWordsDirect(
    bytes32 keyHash,
    uint32 callbackGasLimit,
    uint32 numWords
)
    public
    payable
    onlyOwner
    returns (uint256 requestId)
{
  requestId = COORDINATOR.requestRandomWordsPayment{value: msg.value}(
    keyHash,
    callbackGasLimit,
    numWords
  );
}
```

This function calls the `requestRandomWordsPayment()` function defined in `COORDINATOR` contract, and passes `keyHash`, `callbackGasLimit`, and `numWords` as arguments.
The payment for service is sent through `msg.value` to the `requestRandomWordsPayment()` in `COORDINATOR` contract.
If the payment is larger than expected payment, exceeding payment is returned to the caller of `requestRandomWordsPayment` function, therefore it requires the user contract to define [`receive()` function](https://docs.soliditylang.org/en/v0.8.16/contracts.html#receive-ether-function) as shown in the top of code listing.
Eventually, it generates a request for random words.

In the section below, you can find more detailed explanation how request for random words using direct payment works.

### Request random words with direct payment (coordinator)

The following function is defined in [`VRFCoordinator` contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/VRFCoordinator.sol).

```Solidity
function requestRandomWordsPayment(
    bytes32 keyHash,
    uint32 callbackGasLimit,
    uint32 numWords
) external payable returns (uint256) {
    uint256 vrfFee = estimateDirectPaymentFee();
    if (msg.value < vrfFee) {
        revert InsufficientPayment(msg.value, vrfFee);
    }

    uint64 accId = Prepayment.createAccount();
    Prepayment.addConsumer(accId, msg.sender);
    bool isDirectPayment = true;
    uint256 requestId = requestRandomWordsInternal(
        keyHash,
        accId,
        callbackGasLimit,
        numWords,
        isDirectPayment
    );
    Prepayment.deposit{value: vrfFee}(accId);

    uint256 remaining = msg.value - vrfFee;
    if (remaining > 0) {
        (bool sent, ) = msg.sender.call{value: remaining}("");
        if (!sent) {
            revert RefundFailure();
        }
    }

    return requestId;
}
```

This function first calculates the fee (`vrfFee`) for the request by calling `estimateDirectPaymentFee()` function.
`isDirectPayment` variable indicates whether the request is created through **Prepayment** or **Direct Payment** method.
Then, it deposits the required fee (`vrfFee`) to the account by calling `Prepayment.deposit(accId)` and passing the fee (`vrfFee`) as value.
If the amount of KLAY passed by `msg.value` to the `requestRandomWordsPayment` is larger than required fee (`vrfFee`), the remaining amount is sent back to the caller using the `msg.sender.call()` method.
Finally, the function returns `requestId` that is generated by the `requestRandomWordsInternal()` function.
