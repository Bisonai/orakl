const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const crypto = require('crypto')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { parseKlay, remove0x } = require('./utils.cjs')
const {
  setupOracle,
  generateVrf,
  deploy: deployVrfCoordinator,
  parseRandomWordsRequestedTx,
  parseRequestCanceledTx
} = require('./VRFCoordinator.utils.cjs')
const { deploy: deployVrfConsumerMock } = require('./VRFConsumerMock.utils.cjs')
const {
  deploy: deployPrepayment,
  addCoordinator,
  createAccount,
  deposit,
  addConsumer,
  withdraw,
  cancelAccount
} = require('./Prepayment.utils.cjs')

const DUMMY_KEY_HASH = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
const SINGLE_WORD = 1
const EMPTY_COMMITMENT = '0x0000000000000000000000000000000000000000000000000000000000000000'

function generateDummyPublicProvingKey() {
  const L = 77
  return crypto
    .getRandomValues(new Uint8Array(L))
    .map((a) => {
      return a % 10
    })
    .reduce((acc, v) => acc + v, '')
}

function validateRandomWordsRequestedEvent(
  tx,
  coordinatorContract,
  keyHash,
  accId,
  maxGasLimit,
  numWords,
  sender,
  isDirectPayment
) {
  let eventIndex
  if (isDirectPayment) {
    expect(tx.events.length).to.be.equal(3)
    eventIndex = 1
  } else {
    expect(tx.events.length).to.be.equal(1)
    eventIndex = 0
  }

  const event = coordinatorContract.interface.parseLog(tx.events[eventIndex])
  expect(event.name).to.be.equal('RandomWordsRequested')
  const {
    keyHash: eKeyHash,
    requestId,
    preSeed,
    accId: eAccId,
    callbackGasLimit: eCallbackGasLimit,
    numWords: eNumWords,
    sender: eSender,
    isDirectPayment: eIsDirectPayment
  } = event.args
  expect(eKeyHash).to.be.equal(keyHash)
  if (!isDirectPayment) {
    expect(eAccId).to.be.equal(accId)
  }
  expect(eCallbackGasLimit).to.be.equal(maxGasLimit)
  expect(eNumWords).to.be.equal(numWords)
  expect(eSender).to.be.equal(sender)
  expect(eIsDirectPayment).to.be.equal(isDirectPayment)

  const blockHash = tx.blockHash
  const blockNumber = tx.blockNumber

  return { requestId, preSeed, accId: eAccId, blockHash, blockNumber }
}

async function testCommitmentBeforeFulfillment(coordinator, signer, requestId) {
  // Request has not been fulfilled yet, therewere we expect the
  // commitment to be non-zero
  const commitment = await coordinator.connect(signer).getCommitment(requestId)
  expect(commitment).to.not.be.equal(EMPTY_COMMITMENT)
}

async function testCommitmentAfterFulfillment(coordinator, signer, requestId) {
  // Request has been fulfilled, therewere the requested
  // commitment must be zero
  const commitment = await coordinator.connect(signer).getCommitment(requestId)
  expect(commitment).to.be.equal(EMPTY_COMMITMENT)
}

async function fulfillRandomWords(
  coordinator,
  registeredOracleSigner,
  notRegisteredOracleSigner,
  pi,
  rc,
  isDirectPayment
) {
  // Random word request cannot be fulfilled by an unregistered oracle
  await expect(
    coordinator.connect(notRegisteredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).to.be.revertedWithCustomError(coordinator, 'NoSuchProvingKey')

  // Registered oracle can submit data back to chain
  const tx = await (
    await coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).wait()

  // However even registered oracle cannot fulfill the request more than once
  await expect(
    coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).to.be.revertedWithCustomError(coordinator, 'NoCorrespondingRequest')

  return tx
}

function validateRandomWordsFulfilledEvent(
  tx,
  coordinator,
  prepayment,
  requestId,
  accId,
  isDirectPayment
) {
  let burnedFeeEventIdx
  let randomWordsFulfilledEventIdx
  if (isDirectPayment) {
    expect(tx.events.length).to.be.equal(3)
    burnedFeeEventIdx = 1
    randomWordsFulfilledEventIdx = 2
  } else {
    expect(tx.events.length).to.be.equal(4)
    burnedFeeEventIdx = 1
    randomWordsFulfilledEventIdx = 3
  }

  // Event: AccountBalanceDecreased
  const accountBalanceDecreasedEvent = prepayment.interface.parseLog(tx.events[0])
  expect(accountBalanceDecreasedEvent.name).to.be.equal('AccountBalanceDecreased')
  const {
    accId: dAccId,
    oldBalance: dOldBalance,
    newBalance: dNewBalance
  } = accountBalanceDecreasedEvent.args
  expect(dAccId).to.be.equal(accId)
  expect(dOldBalance).to.be.above(dNewBalance)

  if (isDirectPayment) {
    expect(dNewBalance).to.be.equal(0)
  } else {
    expect(dNewBalance).to.be.above(0)
  }

  // Event: FeeBurned
  const burnedFeeEvent = prepayment.interface.parseLog(tx.events[burnedFeeEventIdx])
  expect(burnedFeeEvent.name).to.be.equal('BurnedFee')
  const { accId: bAccId, amount: bAmount } = burnedFeeEvent.args
  expect(bAccId).to.be.equal(accId)
  expect(bAmount).to.be.above(0)

  // Event: RandomWordsFulfilled
  const randomWordsFulfilledEvent = coordinator.interface.parseLog(
    tx.events[randomWordsFulfilledEventIdx]
  )
  expect(randomWordsFulfilledEvent.name).to.be.equal('RandomWordsFulfilled')
  const {
    requestId: fRequestId,
    // outputSeed: fOutputSeed,
    payment: fPayment,
    success: fSuccess
  } = randomWordsFulfilledEvent.args

  expect(fRequestId).to.be.equal(requestId)
  expect(fSuccess).to.be.equal(true)
  expect(fPayment).to.be.above(0)
}

async function createSigners() {
  let { deployer, consumer, consumer1, consumer2, vrfOracle0 } = await hre.getNamedAccounts()

  const account0 = await ethers.getSigner(deployer)
  const account1 = await ethers.getSigner(consumer)
  const account2 = await ethers.getSigner(consumer1)
  const account3 = await ethers.getSigner(consumer2)
  const account4 = await ethers.getSigner(vrfOracle0)

  return {
    account0,
    account1,
    account2,
    account3,
    account4
  }
}

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2,
    account3,
    account4: protocolFeeRecipient
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner
  }

  // VRFCoordinator
  const coordinatorContract = await deployVrfCoordinator(prepaymentContract.address, deployerSigner)
  expect(await coordinatorContract.typeAndVersion()).to.be.equal('VRFCoordinator v0.1')
  const coordinator = {
    contract: coordinatorContract,
    signer: deployerSigner
  }

  // VRFConsumerMock
  const consumerContract = await deployVrfConsumerMock(coordinatorContract.address, consumerSigner)
  const consumer = {
    contract: consumerContract,
    signer: consumerSigner
  }

  return {
    prepayment,
    coordinator,
    consumer,
    account2,
    account3
  }
}

describe('VRF contract', function () {
  it('Register oracle', async function () {
    const { coordinator } = await loadFixture(deploy)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Registration
    const txReceipt = await (
      await coordinator.contract.registerOracle(oracle, publicProvingKey)
    ).wait()

    expect(txReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinator.contract.interface.parseLog(txReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')

    expect(registerEvent.args['oracle']).to.be.equal(oracle)
    expect(registerEvent.args['keyHash']).to.not.be.undefined
  })

  it('Single oracle cannot be registered more than once, but keyhash can be registered multiple times', async function () {
    const { coordinator } = await loadFixture(deploy)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()
    const publicProvingKey1 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    const publicProvingKey2 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    expect(oracle1).to.not.be.equal(oracle2)
    expect(publicProvingKey1).to.not.be.equal(publicProvingKey2)

    // Registration
    await (await coordinator.contract.registerOracle(oracle1, publicProvingKey1)).wait()
    // Neither oracle or public proving key can be registered twice
    await expect(
      coordinator.contract.registerOracle(oracle1, publicProvingKey1)
    ).to.be.revertedWithCustomError(coordinator.contract, 'OracleAlreadyRegistered')

    // Oracle cannot be registered twice
    await expect(
      coordinator.contract.registerOracle(oracle1, publicProvingKey2)
    ).to.be.revertedWithCustomError(coordinator.contract, 'OracleAlreadyRegistered')

    // Public proving key can be registered twice
    await coordinator.contract.registerOracle(oracle2, publicProvingKey1)

    // There should be single key hash even though we registered
    // oracle twice with the same keyhash
    const [, keyHashesBeforeDeregistration] = await coordinator.contract.getRequestConfig()
    expect(keyHashesBeforeDeregistration.length).to.be.equal(1)

    // Deregister the oracle1
    await coordinator.contract.deregisterOracle(oracle1)

    // There should still be the same single keyhash after the first deregistered oracle
    const [, keyHashesAfterDeregistration] = await coordinator.contract.getRequestConfig()
    expect(keyHashesAfterDeregistration.length).to.be.equal(1)

    // Deregister the oracle2
    await coordinator.contract.deregisterOracle(oracle2)

    // Now, there is not registered oracle, therefore there should also be no keyHash
    const [, keyHashesAfterDeregistration2] = await coordinator.contract.getRequestConfig()
    expect(keyHashesAfterDeregistration2.length).to.be.equal(0)
  })

  it('Deregister registered oracle', async function () {
    const { coordinator } = await loadFixture(deploy)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Cannot deregister underegistered oracle
    await expect(coordinator.contract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinator.contract,
      'NoSuchOracle'
    )

    // Registration
    const tx = await (await coordinator.contract.registerOracle(oracle, publicProvingKey)).wait()
    expect(tx.events.length).to.be.equal(1)
    const event = coordinator.contract.interface.parseLog(tx.events[0])
    expect(event.name).to.be.equal('OracleRegistered')
    let { keyHash: registeredKeyHash } = event.args
    expect(registeredKeyHash).to.not.be.undefined

    // Deregistration
    {
      const tx = await (await coordinator.contract.deregisterOracle(oracle)).wait()
      expect(tx.events.length).to.be.equal(1)
      const event = coordinator.contract.interface.parseLog(tx.events[0])
      expect(event.name).to.be.equal('OracleDeregistered')
      expect(event.args['oracle']).to.be.equal(oracle)
      expect(event.args['keyHash']).to.be.equal(registeredKeyHash)
    }

    // Cannot deregister the same oracle twice
    await expect(coordinator.contract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinator.contract,
      'NoSuchOracle'
    )
  })

  it('requestRandomWords revert on InvalidKeyHash', async function () {
    const { prepayment, coordinator, consumer } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = vrfConfig()
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await expect(
      consumer.contract.requestRandomWords(DUMMY_KEY_HASH, accId, callbackGasLimit, SINGLE_WORD)
    ).to.be.revertedWithCustomError(coordinator.contract, 'InvalidKeyHash')
  })

  it('requestRandomWordsDirect should revert on InvalidKeyHash', async function () {
    const { coordinator, consumer } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = vrfConfig()
    await expect(
      consumer.contract.requestRandomWordsDirectPayment(
        DUMMY_KEY_HASH,
        callbackGasLimit,
        SINGLE_WORD,
        {
          value: parseKlay(1)
        }
      )
    ).to.be.revertedWithCustomError(coordinator.contract, 'InvalidKeyHash')
  })

  it('requestRandomWords can be called by onlyOwner', async function () {
    const { prepayment, consumer, account2: nonOwner } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = vrfConfig()
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await expect(
      consumer.contract
        .connect(nonOwner)
        .requestRandomWords(DUMMY_KEY_HASH, accId, callbackGasLimit, SINGLE_WORD)
    ).to.be.revertedWithCustomError(consumer.contract, 'OnlyOwner')
  })

  it('requestRandomWords with [regular] account', async function () {
    // VRF is a paid service that requires a payment through a
    // Prepayment smart contract. Every [regular] account has to have at
    // least `sMinBalance` in their account in order to succeed with
    // VRF request.
    const {
      consumer,
      account2: oracle,
      account3: unregisteredOracle,
      coordinator,
      prepayment
    } = await loadFixture(deploy)

    // Prepare cordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    const { maxGasLimit: callbackGasLimit, keyHash } = vrfConfig()
    await expect(
      consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).to.be.revertedWithCustomError(coordinator.contract, 'InsufficientPayment')

    // Deposit 2 $KLAY to account with zero balance
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(2))

    // After depositing minimum account to account, we are able to
    // request random words.
    const txRequestRandomWords = await (
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).wait()

    const isDirectPayment = false
    const numWords = SINGLE_WORD
    const sender = consumer.contract.address
    const { requestId, preSeed, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinator.contract,
      keyHash,
      accId,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )

    await testCommitmentBeforeFulfillment(coordinator.contract, consumer.signer, requestId)
    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      numWords
    )

    const txFulfillRandomWords = await fulfillRandomWords(
      coordinator.contract,
      oracle,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment
    )

    await testCommitmentAfterFulfillment(coordinator.contract, consumer.signer, requestId)

    validateRandomWordsFulfilledEvent(
      txFulfillRandomWords,
      coordinator.contract,
      prepayment.contract,
      requestId,
      accId,
      isDirectPayment
    )
  })

  it('requestRandomWords with [temporary] account', async function () {
    const {
      consumer,
      coordinator,
      prepayment,
      account2: oracle,
      account3: unregisteredOracle
    } = await loadFixture(deploy)

    const { maxGasLimit: callbackGasLimit, keyHash } = vrfConfig()

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Request random words through temporary account
    const txRequestRandomWords = await (
      await consumer.contract.requestRandomWordsDirectPayment(
        keyHash,
        callbackGasLimit,
        SINGLE_WORD,
        {
          value: parseKlay('1')
        }
      )
    ).wait()

    const isDirectPayment = true
    const numWords = SINGLE_WORD
    const sender = consumer.contract.address
    const { requestId, preSeed, accId, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinator.contract,
      keyHash,
      0,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )

    await testCommitmentBeforeFulfillment(coordinator.contract, consumer.signer, requestId)
    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      numWords
    )

    const txFulfillRandomWords = await fulfillRandomWords(
      coordinator.contract,
      oracle,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment
    )
    return

    await testCommitmentAfterFulfillment(coordinator.contract, consumer.signer, requestId)

    validateRandomWordsFulfilledEvent(
      txFulfillRandomWords,
      coordinator.contract,
      prepayment.contract,
      requestId,
      accId,
      isDirectPayment
    )
  })

  it('Cancel random words request for [regular] account', async function () {
    const { prepayment, consumer, coordinator, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Request Random Words
    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const txRequestRandomWords = await (
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).wait()

    const requestedRandomWordsEvent = coordinator.contract.interface.parseLog(
      txRequestRandomWords.events[0]
    )
    expect(requestedRandomWordsEvent.name).to.be.equal('RandomWordsRequested')

    const { requestId } = requestedRandomWordsEvent.args

    // Cancel Request
    {
      const tx = await (await consumer.contract.cancelRequest(requestId)).wait()
      const { requestId: _requestId } = parseRequestCanceledTx(coordinator.contract, tx)
      expect(requestId).to.be.equal(_requestId)
    }
  })

  it('Increase nonce by every request with [regular] account', async function () {
    const { prepayment, consumer, coordinator, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()

    // Before first request
    {
      const nonce = await prepayment.contract.getNonce(accId, consumer.contract.address)
      expect(nonce).to.be.equal(1)
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    }

    // After first request
    {
      const nonce = await prepayment.contract.getNonce(accId, consumer.contract.address)
      expect(nonce).to.be.equal(2)
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    }

    // After second request
    {
      const nonce = await prepayment.contract.getNonce(accId, consumer.contract.address)
      expect(nonce).to.be.equal(3)
    }
  })

  it('Increase reqCount by every request with [regular] account', async function () {
    const { prepayment, consumer, coordinator, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Before first request, `reqCount` should be 0
    {
      const reqCount = await prepayment.contract.getReqCount(accId)
      expect(reqCount).to.be.equal(0)
    }

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const txRequestRandomWords = await (
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).wait()

    // The `reqCount` after the request does not change. It gets
    // updated during `chargeFee` call inside of `Account` contract.
    {
      const reqCount = await prepayment.contract.getReqCount(accId)
      expect(reqCount).to.be.equal(0)
    }

    const isDirectPayment = false
    const numWords = SINGLE_WORD
    const sender = consumer.contract.address
    const { requestId, preSeed, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinator.contract,
      keyHash,
      accId,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )

    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      numWords
    )

    await coordinator.contract.connect(oracle).fulfillRandomWords(pi, rc, isDirectPayment)

    // The value of `reqCount` should increase
    const reqCountAfterFulfillment = await prepayment.contract.getReqCount(accId)
    expect(reqCountAfterFulfillment).to.be.equal(1)
  })

  it('Withdraw from account / Cancel account: pending request exists', async function () {
    const { consumer, prepayment, coordinator, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    const amount = parseKlay(1)
    await deposit(prepayment.contract, consumer.signer, accId, amount)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Request
    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const txRequest = await (
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).wait()
    const { requestId } = parseRandomWordsRequestedTx(coordinator.contract, txRequest)

    // Cannot withdraw when pending request exists
    await expect(
      prepayment.contract.connect(consumer.signer).withdraw(accId, amount)
    ).to.be.revertedWithCustomError(prepayment.contract, 'PendingRequestExists')

    // Cannot cancel account when pending request exists
    await expect(
      prepayment.contract.connect(consumer.signer).cancelAccount(accId, consumer.signer.address)
    ).to.be.revertedWithCustomError(prepayment.contract, 'PendingRequestExists')

    // Cancel request
    const txCancelRequest = await (await consumer.contract.cancelRequest(requestId)).wait()
    parseRequestCanceledTx(coordinator.contract, txCancelRequest)

    // Now, we can withdraw
    const { oldBalance, newBalance } = await withdraw(
      prepayment.contract,
      consumer.signer,
      accId,
      amount
    )
    expect(oldBalance).to.be.gt(newBalance)
    expect(newBalance).to.be.equal(0)

    // And also cancel account
    await cancelAccount(prepayment.contract, consumer.signer, accId, consumer.signer.address)
  })

  it('IncorrectCommitment', async function () {
    const { coordinator, consumer, prepayment, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Request
    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const txRequest = await (
      await consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).wait()
    const { preSeed, blockHash, blockNumber, sender, numWords } = parseRandomWordsRequestedTx(
      coordinator.contract,
      txRequest
    )

    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber + 1, // Leads to IcorrectCommitment!
      accId,
      callbackGasLimit,
      sender,
      numWords
    )

    const isDirectPayment = false
    await expect(
      coordinator.contract.connect(oracle).fulfillRandomWords(pi, rc, isDirectPayment)
    ).to.be.revertedWithCustomError(coordinator.contract, 'IncorrectCommitment')
  })

  it('InvalidConsumer', async function () {
    const {
      coordinator,
      consumer,
      prepayment,
      account2: oracle,
      account3: fakeConsumer
    } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    const amount = parseKlay(1)
    await deposit(prepayment.contract, consumer.signer, accId, amount)
    await addConsumer(prepayment.contract, consumer.signer, accId, fakeConsumer.address)

    // Request
    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    await expect(
      consumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, SINGLE_WORD)
    ).to.be.revertedWithCustomError(coordinator.contract, 'InvalidConsumer')
  })

  it('GasLimitTooBig', async function () {
    const { coordinator, consumer, prepayment, account2: oracle } = await loadFixture(deploy)

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    const amount = parseKlay(1)
    await deposit(prepayment.contract, consumer.signer, accId, amount)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Request
    const { keyHash, maxGasLimit } = vrfConfig()
    const tooBigCallbackGasLimit = maxGasLimit + 1
    await expect(
      consumer.contract.requestRandomWords(keyHash, accId, tooBigCallbackGasLimit, SINGLE_WORD)
    ).to.be.revertedWithCustomError(coordinator.contract, 'GasLimitTooBig')
  })

  // TODO send more $KLAY for direct payment
  // TODO getters
})
