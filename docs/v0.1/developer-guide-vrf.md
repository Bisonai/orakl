# Documentation

## Developer guide on how to use VRF on Klaytn

1. what is VRF
A Verifiable Random Function (VRF) is a cryptographic function that generates a random value, or output, based on some input data (called the "seed"). Importantly, the VRF output is verifiable, meaning that anyone who has access to the VRF output and the seed can verify that the output was generated correctly and truly randomly.

In the context of the blockchain, VRFs can be used to provide a source of randomness that is unpredictable and unbiased. This can be useful in various decentralized applications (dApps) that require randomness as a key component, such as in randomized auctions or as part of a decentralized game.

Orakl is a decentralized oracle network that allows smart contracts to securely access off-chain data and other resources. VRF services on [our name] allow smart contracts to use the VRF function to generate verifiably random values, which can be used in various dApps that require randomness.

2. Prequisite: Create and fund your acount (recommended)
You can interact with the prepayment contract to manage your accounts on Orakl.
++ Main functions: should be done before making a request
+ Create account
```
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
This function creates a new account by incrementing a global variable `s_currentAccId` by 1 and storing the value in a local variable `currentAccId`. It then creates an empty array of addresses called consumers and assigns it to the consumers field of `s_accountConfigs[currentAccId]`. It also creates `s_accounts[currentAccId]` with balance and reqCount set to 0. Finally, it emits an event `AccountCreated` with the current account ID and the sender's address as arguments, and returns the current account ID.

+ deposit
```
function deposit(uint64 accId) external payable {
        if (msg.sender.balance < msg.value) {
            revert InsufficientConsumerBalance();
        }
        uint256 amount = msg.value;
        uint256 oldBalance = s_accounts[accId].balance;
        s_accounts[accId].balance += amount;
        s_totalBalance += amount;
        emit AccountBalanceIncreased(accId, oldBalance, oldBalance + amount);
    }

```
This function retrieves the current balance of the account, adds the deposit amount to it. It also updates the total balance of all accounts by adding the deposit amount to the `s_totalBalance` variable. Finally, it emits an event `AccountBalanceIncreased` with the account ID, old balance and new balance as arguments.
+ add consumer

```
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
This function will update the `s_consumers[consumer][accId]` with the value 1, indicating that the consumer is allocated. It will then push the consumer address to the `s_accountConfigs[accId].consumers` array. Finally, it will emit an event AccountConsumerAdded with the account ID and consumer address as arguments.
++ other functions:
+ Transfer ownership of an account
```
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

This function will update the `s_accountConfigs[accId].requestedOwner` with the value of newOwner and emit an event AccountOwnerTransferRequested with the account ID, the current owner and the new owner address as arguments. This event indicates that a request for owner transfer has been made for the account.
+ Accept account ownership
```
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
This function will update the `s_accountConfigs[accId].owner` with the value of the msg.sender which is the address of the new owner and set the `s_accountConfigs[accId].requestedOwner` to `address(0)`. Finally it will emit an event AccountOwnerTransferred with the account ID, the old owner and the new owner address as arguments. This event indicates that the transfer of ownership has been completed for the account.
+ Remove a consumer

```
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

this functions will iterate over the `s_accountConfigs[accId].consumers` array to find the index of the consumer that needs to be removed. It will then swap the last element with the element to be removed and pop out the last element.
It then delete the consumer from the `s_consumers[consumer][accId]` variable and emits an event `AccountConsumerRemoved` with the account ID and consumer address as arguments.
+ Cancel an account

```
function cancelAccount(uint64 accId, address to) external onlyAccOwner(accId) {
        if (pendingRequestExists(accId)) {
            revert PendingRequestExists();
        }
        cancelAccountHelper(accId, to);
    }

```

This function checks if there are any pending requests for the account by calling a function `pendingRequestExists(accId)`. If there are any pending requests, the function will revert with the error message `PendingRequestExists()`.If there are no pending requests, it calls another function `cancelAccountHelper(accId, to)` which will be responsible for canceling the account.
By checking if there are any pending requests before canceling the account, it ensures that the account is not being cancelled in the middle of any important process.

+ withdraw funds from an account

```
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

This function will subtract the amount to be withdrawn from the account's balance and will transfer the withdrawn amount to the owner of the account using the `msg.sender.call` method. Finally, it will emit an event `AccountBalanceDecreased` with the account ID, old balance and new balance as arguments. This event will indicate that the withdrawal has been completed.

3. Request a random word with prepayment method

Check out the example here: VRFConsumerMock.sol [github link]

++ Main functions

+ Request a random word

```
function requestRandomWords(
      bytes32 keyHash,
      uint64 accId,
      uint16 requestConfirmations,
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
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

```
Arguments:

`keyHash`: a bytes32 value representing the hash of the key used to generate the random words, also used to choose a trusted VRF provider.
`accId`: a uint64 value representing the ID of the account associated with the request.
`requestConfirmations`: a uint16 value representing the number of confirmations required for the request.The bigger value, the more secure the random value is. It must be greater than the `minimumRequestBlockConfirmations` and smaller than the `maximumRequestBlockConfirmations` on the coordinator contract.
`callbackGasLimit`: a uint32 value representing the gas limit for the callback function that will be executed after the confirmations have been received.
numWords: a uint32 value representing the number of random words requested.

This function calls the `requestRandomWords()` function from the COORDINATOR contract, passing in the keyHash, accId, requestConfirmations, callbackGasLimit, and numWords as arguments. This will generate the request for random words.
Finally, it will return the requestId that is generated by the `requestRandomWords()` function in the COORDINATOR contract.

+ fulfill a random word

```
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
arguments:

`requestId`: a uint256 value representing the ID of the request
`randomWords`: an array of uint256 values representing the random words generated in response to the request

This function is called by the COORDINATOR contract to generate a random value between 1 and 50 by taking the first element of the randomWords array modulo 50 and adding 1 to it. The result is stored in the storage variable `s_randomResult`.


4. Request random word direct payment

+ In consumer contract:

```
function requestRandomWordsDirect(
      bytes32 keyHash,
      uint16 requestConfirmations,
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
      requestConfirmations,
      callbackGasLimit,
      numWords
    );
  }

```
This function calls the `requestRandomWordsPayment()` function from the COORDINATOR contract, passing in the `keyHash`, `requestConfirmations`, `callbackGasLimit`, and `numWords` as arguments. This will generate the request for random words. It also sends the value in `msg.value` to the `requestRandomWordsPayment()` in COORDINATOR contract

+ `requestRandomWordsPayment()` in COORDINATOR contract

```
function requestRandomWordsPayment(
        bytes32 keyHash,
        uint16 requestConfirmations,
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
            requestConfirmations,
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
This function first calculates the fee for the request by calling `estimateDirectPaymentFee()` function
Boolean `isDirectPayment` is to indicate that this is a direct payment request.
The function then deposits the fee in the account by calling `Prepayment.deposit(accId)` passing the fee as value.
It then calculates the remaining amount from `msg.value` after subtracting the fee and sends the remaining amount back to the caller using the `msg.sender.call()` method.
Finally, it will return the requestId that is generated by the `requestRandomWordsInternal()` function.


