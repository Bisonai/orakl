const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const crypto = require('crypto')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { parseKlay, remove0x } = require('./utils.cjs')
const { State } = require('./State.utils.cjs')
const { setupOracle, generateVrf } = require('./VRFCoordinator.utils.cjs')

const DUMMY_KEY_HASH = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
const NUM_WORDS = 1
const EMPTY_COMMITMENT = '0x0000000000000000000000000000000000000000000000000000000000000000'

async function createSigners() {
  let { deployer, consumer, consumer1, vrfOracle0 } = await hre.getNamedAccounts()

  const deployerSigner = await ethers.getSigner(deployer)
  const consumerSigner = await ethers.getSigner(consumer)
  const consumer1Signer = await ethers.getSigner(consumer)
  const vrfOracle0Signer = await ethers.getSigner(vrfOracle0)

  return {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    vrfOracle0Signer
  }
}

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
  unregisteredOracleSigner,
  pi,
  rc,
  isDirectPayment
) {
  // Random word request cannot be fulfilled by an unregistered oracle
  await expect(
    coordinator.connect(unregisteredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
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

async function deployFixture() {
  const {
    deployer,
    consumer,
    consumer1: sProtocolFeeRecipient,
    consumer2,
    vrfOracle0
  } = await hre.getNamedAccounts()

  // Prepayment
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployer
  })
  prepaymentContract = await prepaymentContract.deploy(sProtocolFeeRecipient)
  await prepaymentContract.deployed()

  // VRFCoordinator
  let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
    signer: deployer
  })
  coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  await coordinatorContract.deployed()

  // VRFConsumerMock
  let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
    signer: consumer
  })
  consumerContract = await consumerContract.deploy(coordinatorContract.address)
  await consumerContract.deployed()

  const coordinatorContractOracleSigner = await ethers.getContractAt(
    'VRFCoordinator',
    coordinatorContract.address,
    vrfOracle0
  )

  // State controller
  const state = new State(consumer, prepaymentContract, consumerContract, coordinatorContract, [
    coordinatorContractOracleSigner
  ])
  await state.initialize('VRFConsumerMock')

  return {
    deployer,
    consumer,
    consumer2,
    vrfOracle0,
    prepaymentContract,
    coordinatorContract,
    consumerContract,

    state
  }
}

describe('VRF contract', function () {
  it('Register oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Registration
    const txReceipt = await (
      await coordinatorContract.registerOracle(oracle, publicProvingKey)
    ).wait()

    expect(txReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')

    expect(registerEvent.args['oracle']).to.be.equal(oracle)
    expect(registerEvent.args['keyHash']).to.not.be.undefined
  })

  it('Single oracle cannot be registered more than once, but keyhash can be registered multiple times', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()
    const publicProvingKey1 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    const publicProvingKey2 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    expect(oracle1).to.not.be.equal(oracle2)
    expect(publicProvingKey1).to.not.be.equal(publicProvingKey2)

    // Registration
    await (await coordinatorContract.registerOracle(oracle1, publicProvingKey1)).wait()
    // Neither oracle or public proving key can be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle1, publicProvingKey1)
    ).to.be.revertedWithCustomError(coordinatorContract, 'OracleAlreadyRegistered')

    // Oracle cannot be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle1, publicProvingKey2)
    ).to.be.revertedWithCustomError(coordinatorContract, 'OracleAlreadyRegistered')

    // Public proving key can be registered twice
    await coordinatorContract.registerOracle(oracle2, publicProvingKey1)

    // There should be single key hash even though we registered
    // oracle twice with the same keyhash
    const [, keyHashesBeforeDeregistration] = await coordinatorContract.getRequestConfig()
    expect(keyHashesBeforeDeregistration.length).to.be.equal(1)

    // Deregister the oracle1
    await coordinatorContract.deregisterOracle(oracle1)

    // There should still be the same single keyhash after the first deregistered oracle
    const [, keyHashesAfterDeregistration] = await coordinatorContract.getRequestConfig()
    expect(keyHashesAfterDeregistration.length).to.be.equal(1)

    // Deregister the oracle2
    await coordinatorContract.deregisterOracle(oracle2)

    // Now, there is not registered oracle, therefore there should also be no keyHash
    const [, keyHashesAfterDeregistration2] = await coordinatorContract.getRequestConfig()
    expect(keyHashesAfterDeregistration2.length).to.be.equal(0)
  })

  it('Deregister registered oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Cannot deregister underegistered oracle
    await expect(coordinatorContract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoSuchOracle'
    )

    // Registration
    const txRegisterReceipt = await (
      await coordinatorContract.registerOracle(oracle, publicProvingKey)
    ).wait()
    expect(txRegisterReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txRegisterReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')
    const kh = registerEvent.args['keyHash']
    expect(kh).to.not.be.undefined

    // Deregistration
    const txDeregisterReceipt = await (await coordinatorContract.deregisterOracle(oracle)).wait()
    expect(txDeregisterReceipt.events.length).to.be.equal(1)
    const deregisterEvent = coordinatorContract.interface.parseLog(txDeregisterReceipt.events[0])
    expect(deregisterEvent.name).to.be.equal('OracleDeregistered')
    expect(deregisterEvent.args['oracle']).to.be.equal(oracle)
    expect(deregisterEvent.args['keyHash']).to.be.equal(kh)

    // Cannot deregister the same oracle twice
    await expect(coordinatorContract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoSuchOracle'
    )
  })

  it('requestRandomWords revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract, state } = await loadFixture(deployFixture)

    const { maxGasLimit } = vrfConfig()
    const accId = await state.createAccount()

    await expect(
      consumerContract.requestRandomWords(DUMMY_KEY_HASH, accId, maxGasLimit, NUM_WORDS)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWordsDirect should revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract } = await loadFixture(deployFixture)

    const { maxGasLimit } = vrfConfig()
    const value = parseKlay(1)

    await expect(
      consumerContract.requestRandomWordsDirectPayment(DUMMY_KEY_HASH, maxGasLimit, NUM_WORDS, {
        value
      })
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWords can be called by onlyOwner', async function () {
    const { consumerContract, consumer2: nonOwnerAddress, state } = await loadFixture(deployFixture)

    const consumerContractNonOwnerSigner = await ethers.getContractAt(
      'VRFConsumerMock',
      consumerContract.address,
      nonOwnerAddress
    )
    const { maxGasLimit } = vrfConfig()
    const accId = await state.createAccount()

    await expect(
      consumerContractNonOwnerSigner.requestRandomWords(
        DUMMY_KEY_HASH,
        accId,
        maxGasLimit,
        NUM_WORDS
      )
    ).to.be.revertedWithCustomError(consumerContractNonOwnerSigner, 'OnlyOwner')
  })

  it('requestRandomWords with [regular] account', async function () {
    // VRF is a paid service that requires a payment through a
    // Prepayment smart contract. Every [regular] account has to have at
    // least `sMinBalance` in their account in order to succeed with
    // VRF request.
    const {
      consumer,
      vrfOracle0,
      coordinatorContract,
      consumerContract,
      prepaymentContract,
      state
    } = await loadFixture(deployFixture)
    const {
      consumerSigner,
      consumer1Signer: unregisteredOracle,
      vrfOracle0Signer
    } = await createSigners()

    const {
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig,
      sk,
      pk,
      pkX,
      pkY,
      publicProvingKey,
      keyHash
    } = vrfConfig()

    await coordinatorContract.registerOracle(vrfOracle0, publicProvingKey)
    await coordinatorContract.setConfig(
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    await state.addCoordinator(coordinatorContract.address)

    const accId = await state.createAccount()
    await state.addConsumer(consumerContract.address)

    await expect(
      consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')

    await state.deposit('2')

    // After depositing minimum account to account, we are able to
    // request random words.
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
    ).wait()

    const isDirectPayment = false
    const callbackGasLimit = maxGasLimit
    const numWords = NUM_WORDS
    const sender = consumerContract.address
    const { requestId, preSeed, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinatorContract,
      keyHash,
      accId,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )

    await testCommitmentBeforeFulfillment(coordinatorContract, consumerSigner, requestId)
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
      coordinatorContract,
      vrfOracle0Signer,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment
    )

    await testCommitmentAfterFulfillment(coordinatorContract, consumerSigner, requestId)

    validateRandomWordsFulfilledEvent(
      txFulfillRandomWords,
      coordinatorContract,
      prepaymentContract,
      requestId,
      accId,
      isDirectPayment
    )
  })

  it('requestRandomWords with [temporary] account', async function () {
    const {
      consumer,
      vrfOracle0,
      coordinatorContract,
      consumerContract,
      prepaymentContract,
      state
    } = await loadFixture(deployFixture)
    const {
      consumerSigner,
      consumer1Signer: unregisteredOracle,
      vrfOracle0Signer
    } = await createSigners()

    const {
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig,
      sk,
      pk,
      pkX,
      pkY,
      publicProvingKey,
      keyHash
    } = vrfConfig()

    await coordinatorContract.registerOracle(vrfOracle0, publicProvingKey)
    await coordinatorContract.setConfig(
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    await state.addCoordinator(coordinatorContract.address)

    // Request random words through temporary account
    const value = parseKlay('1')
    const callbackGasLimit = maxGasLimit
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWordsDirectPayment(keyHash, callbackGasLimit, NUM_WORDS, {
        value
      })
    ).wait()

    const isDirectPayment = true
    const numWords = NUM_WORDS
    const sender = consumerContract.address
    const { requestId, preSeed, accId, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinatorContract,
      keyHash,
      0,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )

    await testCommitmentBeforeFulfillment(coordinatorContract, consumerSigner, requestId)
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
      coordinatorContract,
      vrfOracle0Signer,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment
    )

    await testCommitmentAfterFulfillment(coordinatorContract, consumerSigner, requestId)

    validateRandomWordsFulfilledEvent(
      txFulfillRandomWords,
      coordinatorContract,
      prepaymentContract,
      requestId,
      accId,
      isDirectPayment
    )
  })

  it('Cancel random words request for [regular] account', async function () {
    const { vrfOracle0, coordinatorContract, consumerContract, state } = await loadFixture(
      deployFixture
    )

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    await setupOracle(coordinatorContract, vrfOracle0)
    await state.addCoordinator(coordinatorContract.address)
    const accId = await state.createAccount()
    await state.addConsumer(consumerContract.address)
    await state.deposit('2')

    // Request Random Words
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, NUM_WORDS)
    ).wait()

    const requestedRandomWordsEvent = coordinatorContract.interface.parseLog(
      txRequestRandomWords.events[0]
    )
    expect(requestedRandomWordsEvent.name).to.be.equal('RandomWordsRequested')

    const { requestId } = requestedRandomWordsEvent.args

    // Cancel Request
    const txCancelRequest = await (await consumerContract.cancelRequest(requestId)).wait()

    const randomWordsRequestCancelledEvent = coordinatorContract.interface.parseLog(
      txCancelRequest.events[0]
    )
    expect(randomWordsRequestCancelledEvent.name).to.be.equal('RequestCanceled')

    const { requestId: cRequestId } = randomWordsRequestCancelledEvent.args
    expect(requestId).to.be.equal(cRequestId)
  })

  it('Cannot cancel an invalid request', async function () {
    const { vrfOracle0, coordinatorContract, consumerContract, state } = await loadFixture(
      deployFixture
    )

    await setupOracle(coordinatorContract, vrfOracle0)
    await state.addCoordinator(coordinatorContract.address)

    const invalidRequestId = 123
    await expect(coordinatorContract.cancelRequest(invalidRequestId)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoCorrespondingRequest'
    )
  })

  it('Increase nonce by every request with [regular] account', async function () {
    const { vrfOracle0, coordinatorContract, consumerContract, state } = await loadFixture(
      deployFixture
    )

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    await setupOracle(coordinatorContract, vrfOracle0)
    await state.addCoordinator(coordinatorContract.address)
    const accId = await state.createAccount()
    await state.addConsumer(consumerContract.address)
    await state.deposit('1')

    // Before first request
    const nonce1 = await state.prepaymentContract.getNonce(accId, consumerContract.address)
    expect(nonce1).to.be.equal(1)
    await consumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, NUM_WORDS)

    // After first request
    const nonce2 = await state.prepaymentContract.getNonce(accId, consumerContract.address)
    expect(nonce2).to.be.equal(2)
    await consumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, NUM_WORDS)

    // After second request
    const nonce3 = await state.prepaymentContract.getNonce(accId, consumerContract.address)
    expect(nonce3).to.be.equal(3)
  })

  it('Increase reqCount by every request with [regular] account', async function () {
    const { vrfOracle0, coordinatorContract, consumerContract, state } = await loadFixture(
      deployFixture
    )
    const { vrfOracle0Signer } = await createSigners()

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    await setupOracle(coordinatorContract, vrfOracle0)
    await state.addCoordinator(coordinatorContract.address)
    const accId = await state.createAccount()
    await state.addConsumer(consumerContract.address)
    await state.deposit('1')

    // Before first request, `reqCount` should be 0
    const reqCountBeforeRequest = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountBeforeRequest).to.be.equal(0)

    const txRequestRandomWords = await (
      await consumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, NUM_WORDS)
    ).wait()

    // The `reqCount` after the request does not change. It gets
    // updated during `chargeFee` call inside of `Account` contract.
    const reqCountAfterRequest = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountAfterRequest).to.be.equal(0)

    const isDirectPayment = false
    const numWords = NUM_WORDS
    const sender = consumerContract.address
    const { requestId, preSeed, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinatorContract,
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

    await coordinatorContract.connect(vrfOracle0Signer).fulfillRandomWords(pi, rc, isDirectPayment)

    // The value of `reqCount` should increase
    const reqCountAfterFulfillment = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountAfterFulfillment).to.be.equal(1)
  })

  // TODO send more $KLAY for direct payment
  // TODO getters
  // TODO pending request exist
})
