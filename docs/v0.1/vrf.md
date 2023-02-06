# Verifiable Random Function (VRF)

## What is Verifiable Random Function?

A Verifiable Random Function (VRF) is a cryptographic function that generates a random value, or output, based on some input data (called the "seed").
Importantly, the VRF output is verifiable, meaning that anyone who has access to the VRF output and the seed can verify that the output was generated correctly.

In the context of the blockchain, VRFs can be used to provide a source of randomness that is unpredictable and unbiased.
This can be useful in various decentralized applications (dApps) that require randomness as a key component, such as in randomized auctions or as part of a decentralized games.

Orakl VRF allows smart contracts to use the VRF function to generate verifiably random values, which can be used in various dApps that require randomness.
Orakl VRF can be requested using two different payment approaches:

* [Prepayment](#prepayment-recommended)
* [Direct Payment](#direct-payment)

**Prepayment** requires user to create an account and deposit KLAY before requesting VRF.
With **Direct Payment**, user can pay for VRF together while requesting VRF.
The details of both approaches are explained below.

## Prepayment (recommended)

### 1. Prerequisite: Create and fund your acount

You can interact with the [Prepayment contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol) to manage your accounts in Orakl.
There are several steps that has to be performed before creating a VRF request:

1. [Create account](#create-account)
2. [Deposit KLAY to account](#deposit-klay-to-account)
3. [Add consumer](#add-consumer)

Prepayment supports many other helpful functions.
In this document, we describe some of them:

* [Transfer account ownership](#transfer-account-ownership)
* [Accept account ownership](#accept-account-ownership)
* [Remove consumer](#remove-consumer)
* [Cancel account](#cancel-account)
* [Withdraw funds from account](#withdraw-funds-from-account)

The functions are described in subsections below.

#### Create account

```Solidity
function createAccount() external returns (uint64) {
    s_currentAccId++;
    uint64 currentAccId = s_currentAccId;
    address[] memory consumers = new address[](0);
    s_accounts[currentAccId] = Account({balance: 0, reqCount: 0});
    s_accountConfigs[currentAccId] = AccountConfig({
        owner: msg.sender,
        requestedOwner: address(0),
        consumers: consumers
    });

    emit AccountCreated(currentAccId, msg.sender);
    return currentAccId;
}
```

This function creates a new account by incrementing a global variable `s_currentAccId` by 1 and storing the value in a local variable `currentAccId`.
Then, it creates an empty array of addresses called consumers and assigns it to the consumers field of `s_accountConfigs[currentAccId]`.
It also creates `s_accounts[currentAccId]` with balance and `reqCount` set to 0.
Information about newly created account ID and sender's address are emitted using `AccountCreated` event.
Finally, it returns the new account ID.

#### Deposit KLAY to account

```Solidity
function deposit(uint64 accId) external payable {
    if (msg.sender.balance < msg.value) {
        revert InsufficientConsumerBalance();
    }
    uint256 amount = msg.value;
    uint256 oldBalance = s_accounts[accId].balance;
    S_accounts[accId].balance += amount;
    s_totalBalance += amount;
    emit AccountBalanceIncreased(accId, oldBalance, oldBalance + amount);
}
```

This function retrieves the current balance of the account, and increases it by given deposit.
It also updates the total balance of all accounts by adding the deposit amount to the `s_totalBalance` variable.
Finally, it emits an event `AccountBalanceIncreased` with the account ID, old balance and new balance as arguments.

#### Add consumer

```Solidity
function addConsumer(uint64 accId, address consumer) external onlyAccOwner(accId) {
    // Already maxed, cannot add any more consumers.
    if (s_accountConfigs[accId].consumers.length >= MAX_CONSUMERS) {
        revert TooManyConsumers();
    }
    if (s_consumers[consumer][accId] != 0) {
        // Idempotence - do nothing if already added.
        // Ensures uniqueness in s_accounts[accId].consumers.
        return;
    }
    // Initialize the nonce to 1, indicating the consumer is allocated.
    s_consumers[consumer][accId] = 1;
    s_accountConfigs[accId].consumers.push(consumer);

    emit AccountConsumerAdded(accId, consumer);
}
```

This function increases the value of `s_consumers[consumer][accId]` by 1, indicating the number of consumer under given `accId`.
Then, it pushes the consumer address to the `s_accountConfigs[accId].consumers` array.
Finally, it emits an event `AccountConsumerAdded` with the account ID and consumer address as arguments.

#### Transfer account ownership

```Solidity
function requestAccountOwnerTransfer(
    uint64 accId,
    address newOwner
) external onlyAccOwner(accId) {
    // Proposing to address(0) would never be claimable so don't need to check.
    if (s_accountConfigs[accId].requestedOwner != newOwner) {
        s_accountConfigs[accId].requestedOwner = newOwner;
        emit AccountOwnerTransferRequested(accId, msg.sender, newOwner);
    }
}
```

This function updates the `s_accountConfigs[accId].requestedOwner` with the value of `newOwner` and emits an event `AccountOwnerTransferRequested` with the account ID, the current owner and the new owner address as arguments.
`AccountOwnerTransferRequested` indicates that a request for owner transfer has been made for the account.

#### Accept account ownership

```Solidity
function acceptAccountOwnerTransfer(uint64 accId) external {
    if (s_accountConfigs[accId].owner == address(0)) {
        revert InvalidAccount();
    }
    if (s_accountConfigs[accId].requestedOwner != msg.sender) {
        revert MustBeRequestedOwner(s_accountConfigs[accId].requestedOwner);
    }
    address oldOwner = s_accountConfigs[accId].owner;
    s_accountConfigs[accId].owner = msg.sender;
    s_accountConfigs[accId].requestedOwner = address(0);
    emit AccountOwnerTransferred(accId, oldOwner, msg.sender);
}
```

This function updates the `s_accountConfigs[accId].owner` with the value of the `msg.sender` which is the address of the new owner and set the `s_accountConfigs[accId].requestedOwner` to `address(0)`.
Finally it emits `AccountOwnerTransferred` event with the account ID, the old owner and the new owner address as arguments.
`AccountOwnerTransferred` indicates that the transfer of ownership has been completed for the account.

#### Remove consumer

```Solidity
function removeConsumer(uint64 accId, address consumer) external onlyAccOwner(accId) {
    if (s_consumers[consumer][accId] == 0) {
        revert InvalidConsumer(accId, consumer);
    }
    // Note bounded by MAX_CONSUMERS
    address[] memory consumers = s_accountConfigs[accId].consumers;
    uint256 lastConsumerIndex = consumers.length - 1;
    for (uint256 i = 0; i < consumers.length; i++) {
        if (consumers[i] == consumer) {
            address last = consumers[lastConsumerIndex];
            // Storage write to preserve last element
            s_accountConfigs[accId].consumers[i] = last;
            // Storage remove last element
            s_accountConfigs[accId].consumers.pop();
            break;
        }
    }
    delete s_consumers[consumer][accId];
    emit AccountConsumerRemoved(accId, consumer);
}
```

This function iterates over the `s_accountConfigs[accId].consumers` array to find the index of the consumer that needs to be removed.
Then, it swaps the last element with the element to be removed and pop out the last element.
Finally, it deletes the consumer from the `s_consumers[consumer][accId]` variable and emits an event `AccountConsumerRemoved` with the account ID and consumer address as arguments.

#### Cancel account

```Solidity
function cancelAccount(uint64 accId, address to) external onlyAccOwner(accId) {
    if (pendingRequestExists(accId)) {
        revert PendingRequestExists();
    }
    cancelAccountHelper(accId, to);
}
```

This function checks if there are any pending requests for the account by calling a function `pendingRequestExists(accId)`.
If there are any pending requests, the function reverts with the error message `PendingRequestExists()`.
If there are no pending requests, it calls another function `cancelAccountHelper(accId, to)` which is responsible for canceling the account.
By checking if there are any pending requests before canceling the account, it ensures that the account is not being cancelled in the middle of any important process.

#### Withdraw funds from account

```Solidity
function withdraw(uint64 accId, uint256 amount) external onlyAccOwner(accId) {
    if (pendingRequestExists(accId)) {
        revert PendingRequestExists();
    }

    uint256 oldBalance = s_accounts[accId].balance;
    if ((oldBalance < amount) || (address(this).balance < amount)) {
        revert InsufficientBalance();
    }

    s_accounts[accId].balance -= amount;

    (bool sent, ) = msg.sender.call{value: amount}("");
    if (!sent) {
        revert InsufficientBalance();
    }

    emit AccountBalanceDecreased(accId, oldBalance, oldBalance - amount);
}
```

This function subtracts the amount to be withdrawn from the account's balance and transfers the withdrawn amount to the owner of the account using the `msg.sender.call` method.
Finally, it emits an event `AccountBalanceDecreased` with the account ID, old balance and new balance as arguments.
`AccountBalanceDecreased` event indicates that the withdrawal has been completed.

### 2. Request & fulfill random words

A detailed example of how to use Orakl VRF can be found at example repository [`vrf-consumer`](https://github.com/Bisonai-CIC/vrf-consumer).
There are two functions that has to be implemented:

* [Request random words](#request-random-words)
* [Fulfill-random words](#fulfill-random-words)


#### Request random words

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

The arguments of `requestRandomWords` function are explained below:

* `keyHash`: a `bytes32` value representing the hash of the key used to generate the random words, also used to choose a trusted VRF provider.
* `accId`: a `uint64` value representing the ID of the account associated with the request.
* `callbackGasLimit`: a `uint32` value representing the gas limit for the callback function that executes after the confirmations have been received.
* `numWords`: a `uint32` value representing the number of random words requested.

The function call `requestRandomWords()` on `COORDINATOR` contract passes `keyHash`, `accId`, `callbackGasLimit`, and `numWords` as arguments.
Execution of this function generates a request for random words that is uniquely defined with `requestId`, a return value from the function call on `COORDINATOR` contract.


#### Fulfill random words

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

The function is called by the `COORDINATOR` contract to generate a random value between 1 and 50 by taking the first element of the `randomWords` array modulo 50 and adding 1 to it.
The result is stored in the storage variable `s_randomResult`.


## Direct Payment

**Direct Payment** represents an alternative payment method which does not require a user to create account, depositor KLAY, and assign consumer before being able to utilize VRF functionality.
Request for VRF with direct payment is a little bit different compared to prepayment, however, fulfillment function is exactly same.

### Request random words

```Solidity
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

This function calls the `requestRandomWordsPayment()` function from the `COORDINATOR` contract, passing in the `keyHash`, `callbackGasLimit`, and `numWords` as arguments.
The payment for service is sent through `msg.value` to the `requestRandomWordsPayment()` in `COORDINATOR` contract.
If the payment is larger than expected payment, exceeding payment is return to the caller of `COORDINATOR` contract.
It requires the consumer the have `receive()` function defined.
Eventually, it generates a request for random words.

In the section below, you can find more detailed explanation how request for random words with direct payment works.


### Fulfill random words

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

This function first calculates the fee for the request by calling `estimateDirectPaymentFee()` function.
Boolean `isDirectPayment` indicates that this request is a direct payment request.
Then it deposits the required fee in the account by calling `Prepayment.deposit(accId)` passing the fee as value.
If the amount of KLAY passed by `msg.value` is larger than required fee (`vrfFee`), the remaining amount is sent back to the caller using the `msg.sender.call()` method.
Finally, it returns `requestId` that is generated by the `requestRandomWordsInternal()` function.
