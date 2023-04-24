const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const oraklVrf = import('@bisonai/orakl-vrf')
const crypto = require('crypto')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { parseKlay, remove0x } = require('./utils.cjs')
const { State } = require('./State.utils.cjs')

const DUMMY_KEY_HASH = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
const NUM_WORDS = 1

function generateDummyPublicProvingKey() {
  const L = 77
  return crypto
    .getRandomValues(new Uint8Array(L))
    .map((a) => {
      return a % 10
    })
    .reduce((acc, v) => acc + v, '')
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

  it('Do not allow to register the same oracle or public proving key twice', async function () {
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

    // Public proving key cannot be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle2, publicProvingKey1)
    ).to.be.revertedWithCustomError(coordinatorContract, 'ProvingKeyAlreadyRegistered')
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

    const minBalance = '0.001'
    await state.setMinBalance(minBalance)

    const accId = await state.createAccount()
    state.addConsumer(consumerContract.address)

    await expect(
      consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')

    state.deposit('2')

    // After depositing minimum account to account, we are able to
    // request random words.
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
    ).wait()
    const blockHash = txRequestRandomWords.blockHash
    const blockNumber = txRequestRandomWords.blockNumber

    expect(txRequestRandomWords.events.length).to.be.equal(1)
    const requestedRandomWordsEvent = coordinatorContract.interface.parseLog(
      txRequestRandomWords.events[0]
    )
    expect(requestedRandomWordsEvent.name).to.be.equal('RandomWordsRequested')

    const {
      keyHash: eKeyHash,
      requestId: eRequestId,
      preSeed: ePreSeed,
      accId: eAccId,
      callbackGasLimit: eCallbackGasLimit,
      numWords: eNumWords,
      sender: eSender,
      isDirectPayment: eIsDirectPayment
    } = requestedRandomWordsEvent.args
    expect(eKeyHash).to.be.equal(keyHash)
    expect(eAccId).to.be.equal(accId)
    expect(eCallbackGasLimit).to.be.equal(maxGasLimit)
    expect(eNumWords).to.be.equal(NUM_WORDS)
    expect(eSender).to.be.equal(consumerContract.address)
    expect(eIsDirectPayment).to.be.equal(false)

    const alpha = remove0x(
      ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [ePreSeed, blockHash])
    )

    // Simulate off-chain proof generation
    const { processVrfRequest } = await oraklVrf
    const { proof, uPoint, vComponents } = processVrfRequest(alpha, {
      sk,
      pk,
      pkX,
      pkY,
      keyHash
    })

    // Oracle submits data back to chain
    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'VRFCoordinator',
      coordinatorContract.address,
      vrfOracle0
    )
    const isDirectPayment = false
    const txFulfillRandomWords = await (
      await coordinatorContractOracleSigner.fulfillRandomWords(
        [publicProvingKey, proof, ePreSeed, uPoint, vComponents],
        [blockNumber, eAccId, eCallbackGasLimit, NUM_WORDS, eSender],
        isDirectPayment
      )
    ).wait()

    // Check the event information //////////////////////////////////////////////
    expect(txFulfillRandomWords.events.length).to.be.equal(4)

    // Event: AccountBalanceDecreased
    const accountBalanceDecreasedEvent = prepaymentContract.interface.parseLog(
      txFulfillRandomWords.events[0]
    )
    expect(accountBalanceDecreasedEvent.name).to.be.equal('AccountBalanceDecreased')

    const {
      accId: dAccId,
      oldBalance: dOldBalance,
      newBalance: dNewBalance
    } = accountBalanceDecreasedEvent.args
    expect(dAccId).to.be.equal(eAccId)
    expect(dOldBalance).to.be.above(dNewBalance)
    expect(dNewBalance).to.be.above(0)

    // Event: FeeBurned
    const burnedFeeEvent = prepaymentContract.interface.parseLog(txFulfillRandomWords.events[1])
    expect(burnedFeeEvent.name).to.be.equal('BurnedFee')
    const { accId: bAccId, amount: bAmount } = burnedFeeEvent.args
    expect(bAccId).to.be.equal(eAccId)
    expect(bAmount).to.be.above(0)

    // Event: RandomWordsFulfilled
    const randomWordsFulfilledEvent = coordinatorContract.interface.parseLog(
      txFulfillRandomWords.events[3]
    )
    expect(randomWordsFulfilledEvent.name).to.be.equal('RandomWordsFulfilled')

    const {
      requestId: fRequestId,
      // outputSeed: fOutputSeed,
      payment: fPayment,
      success: fSuccess
    } = randomWordsFulfilledEvent.args

    expect(fRequestId).to.be.equal(eRequestId)
    expect(fSuccess).to.be.equal(true)
    expect(fPayment).to.be.above(0)
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

    // Set a direct payment fee
    const setFulfillmentFee = parseKlay(1)
    const setBaseFee = parseKlay(0.25)
    const setConfig = [setFulfillmentFee, setBaseFee]
    const txSetDirectPaymentConfig = await (
      await coordinatorContract.setDirectPaymentConfig(setConfig)
    ).wait()

    expect(txSetDirectPaymentConfig.events.length).to.be.equal(1)
    const directPaymentConfigSetEvent = coordinatorContract.interface.parseLog(
      txSetDirectPaymentConfig.events[0]
    )
    expect(directPaymentConfigSetEvent.name).to.be.equal('DirectPaymentConfigSet')

    const { fulfillmentFee, baseFee } = directPaymentConfigSetEvent.args
    expect(fulfillmentFee).to.be.equal(setFulfillmentFee)
    expect(baseFee).to.be.equal(setBaseFee)

    const [getFulfillmentFee, getBaseFee] = await coordinatorContract.getDirectPaymentConfig()
    const value = ethers.BigNumber.from(fulfillmentFee).add(baseFee)

    // Request random words through temporary account
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWordsDirectPayment(keyHash, maxGasLimit, NUM_WORDS, {
        value
      })
    ).wait()

    expect(txRequestRandomWords.events.length).to.be.equal(3)
    const requestedRandomWordsEvent = coordinatorContract.interface.parseLog(
      txRequestRandomWords.events[1]
    )
    expect(requestedRandomWordsEvent.name).to.be.equal('RandomWordsRequested')
  })

  it('cancel random words request for [regular] account', async function () {
    const {
      consumer,
      vrfOracle0,
      coordinatorContract,
      consumerContract,
      prepaymentContract,
      state
    } = await loadFixture(deployFixture)

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
    state.addConsumer(consumerContract.address)

    // Request Random Words
    const txRequestRandomWords = await (
      await consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
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
    expect(randomWordsRequestCancelledEvent.name).to.be.equal('RandomWordsRequestCanceled')

    const { requestId: cRequestId } = randomWordsRequestCancelledEvent.args
    expect(requestId).to.be.equal(cRequestId)
  })

  // TODO send more $KLAY for direct payment
  // TODO fulfill direct payment request
  // TODO getters
})
