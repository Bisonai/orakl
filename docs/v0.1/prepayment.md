# Prepayment

## What is Prepayment?

**Prepayment** is one of the payment solutions for Orakl Network.
It is implemented within [`Prepayment` smart contract](https://github.com/Bisonai-CIC/orakl/blob/master/contracts/src/v0.1/Prepayment.sol) and currently it can be used as a payment for [Verifiable Random Function (VRF)](vrf.md) and [Request-Response](request-response.md).
You can read about other supported payment solutions at [Payment section of the Developer's guide](readme.md#payment).

## How to use Prepayment?

The main components of **Prepayment** are **Account**, **Account Owner**, **Consumer** and **Coordinator**.

**Account Owners** are entities that create an **Account** (`createAccount`).
They can also close the **Account** (`cancelAccount`), add (`addConsumer`) or remove **Consumer** (`removeConsumer`) from their **Account(s)**.
KLAY can be withdrawn from account only by the **Account Owner**, however anybody is allowed to deposit (`deposit`) KLAY to any account.
**Consumers** assigned to **Account** use the account balance to pay for Orakl Network services.
The ownership of account can be transfered to other entity through a two-step process (`requestAccountOwnerTransfer`, `acceptAccountOwnerTransfer`).
**Coordinators** are smart contracts that can fulfill request issued by **Consumers**, and they are rewarded for their work (`chargeFee`).
Consequently, they can withdraw their earnings (`nodeWidthdraw`).
**Coordinators** can be added (`addCoordinator`) or removed (`removeCoordinator`) only by the owner of `Prepayment` smart contract.

* [Prerequisites](#prerequisites)
* [Other functions](#other-functions)

There are several steps that has to be performed before being able to use **Prepayment** for Orakl Network services.
The list of required step is shown below:

### Prerequisites

1. [Create account](#create-account)
2. [Deposit KLAY to account](#deposit-klay-to-account)
3. [Add consumer](#add-consumer)

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

### Other functions

Prepayment supports many other helpful functions.
In this document, we describe some of them:

* [Transfer account ownership](#transfer-account-ownership)
* [Accept account ownership](#accept-account-ownership)
* [Remove consumer](#remove-consumer)
* [Cancel account](#cancel-account)
* [Withdraw funds from account](#withdraw-funds-from-account)

The functions are described in subsections below.

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
