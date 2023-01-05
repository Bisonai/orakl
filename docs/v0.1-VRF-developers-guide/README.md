# Documentation

## Developer guide on how to use VRF on Klaytn

1. what is VRF
A Verifiable Random Function (VRF) is a cryptographic function that generates a random value, or output, based on some input data (called the "seed"). Importantly, the VRF output is verifiable, meaning that anyone who has access to the VRF output and the seed can verify that the output was generated correctly and truly randomly.

In the context of the blockchain, VRFs can be used to provide a source of randomness that is unpredictable and unbiased. This can be useful in various decentralized applications (dApps) that require randomness as a key component, such as in randomized auctions or as part of a decentralized game.

[our name] is a decentralized oracle network that allows smart contracts to securely access off-chain data and other resources. VRF services on [our name] allow smart contracts to use the VRF function to generate verifiably random values, which can be used in various dApps that require randomness.

2. Request a random word

```
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.16;

import '../VRFConsumerBase.sol';
import '../interfaces/VRFCoordinatorInterface.sol';


contract VRFConsumerMock is VRFConsumerBase {
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
      // ConfirmedOwner(msg.sender) TODO
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
The VRFConsumerMock contract has a few different functions:

requestRandomWords(): This function initiates a request to the VRF coordinator for a specified number of random words. It specifies the key hash, accId, request confirmations, callback gas limit, and number of words to be generated.
fulfillRandomWords(): This function is called by the VRF coordinator when the requested random words are available. It stores the first random word in the s_randomResult storage variable and modulates it by 50, adding 1 to the result.
The VRFConsumerMock contract also has a constructor function that sets the contract owner and the VRF coordinator address, and a onlyOwner() modifier that can be used to restrict certain functions to be called only by the contract owner.

This version of the VRFConsumerMock contract does not include the requestRandomWordsDirect() function, which allows the caller to specify a payment amount in Ether.


3. Request random word direct payment



4. the prepayment contract: create account, fund your account, add consomers, and more
[link]
The Prepayment contract is a contract that allows users to create and manage prepaid accounts that can be used to pay for external resources, such as API calls or other data.

The contract includes functions for creating and managing prepaid accounts, such as:

createAccount(): This function is used to create a new prepaid account.
fundAccount(): This function is used to add funds to an existing prepaid account.
decreaseAccount(): This function is used to decrease the balance of an existing prepaid account.
cancelAccount(): This function is used to cancel an existing prepaid account and withdraw any remaining balance to the owner of the account.
The contract also includes functions for managing the consumers (i.e., external entities) that are authorized to use the prepaid accounts, such as:

addConsumer(): This function is used to add a new consumer to an existing prepaid account.
removeConsumer(): This function is used to remove an existing consumer from a prepaid account.
The contract also includes functions for safely transferring the ownership of a prepaid account, such as:

requestAccountOwnerTransfer(): This function is used to request the transfer of ownership of a prepaid account to a new owner.
transferAccountOwner(): This function is used to complete the transfer of ownership of a prepaid account.

5. VRFCoordinator interface
[link]
he VRFCoordinatorInterface includes the following functions:

getRequestConfig(): This function returns the minimum number of confirmations required for requests, the maximum gas limit for requests, and a list of registered key hashes.

requestRandomWords(): This function is used by requesting contracts to initiate a request for a specified number of random values. It takes as input the key hash (corresponding to a particular oracle job), the ID of the VRF subscription, the minimum number of confirmations required for the request, the callback gas limit (the amount of gas the requesting contract would like to receive in the fulfillRandomWords callback), and the number of random values requested. It returns a unique request ID.

requestRandomWordsPayment(): This function is similar to requestRandomWords(), but it also allows the requesting contract to specify a payment amount in Klay. This payment is used to pay for the cost of generating the random values.

6. VRF coordinator
[link]

