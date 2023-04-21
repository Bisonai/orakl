const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { State } = require('./State.utils.cjs')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const { parseKlay } = require('./utils.cjs')
const { median: medianFn } = require('mathjs')

const EVENT_ARGS = [
  'requestId',
  'jobId',
  'accId',
  'callbackGasLimit',
  'sender',
  'isDirectPayment',
  'data'
]

async function deployFixture() {
  const {
    deployer,
    consumer,
    rrOracle0,
    consumer1: sProtocolFeeRecipient
  } = await hre.getNamedAccounts()
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig, directFeeConfig } =
    requestResponseConfig()

  // Prepayment
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployer
  })
  prepaymentContract = await prepaymentContract.deploy(sProtocolFeeRecipient)
  await prepaymentContract.deployed()

  // RequestResponseCoordinator
  let coordinatorContract = await ethers.getContractFactory('RequestResponseCoordinator', {
    signer: deployer
  })
  coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  await coordinatorContract.deployed()

  // RequestResponseConsumerMock
  let consumerContract = await ethers.getContractFactory('RequestResponseConsumerMock', {
    signer: consumer
  })
  consumerContract = await consumerContract.deploy(coordinatorContract.address)
  await consumerContract.deployed()

  const coordinatorContractOracleSigner = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle0
  )

  // State controller
  const state = new State(
    consumer,
    prepaymentContract,
    consumerContract,
    coordinatorContract,
    coordinatorContractOracleSigner
  )
  await state.initialize('RequestResponseConsumerMock')
  await state.setMinBalance('0.001')
  await state.addCoordinator(coordinatorContract.address)

  return {
    deployer,
    consumer,
    rrOracle0,

    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
    directFeeConfig,
    prepaymentContract,
    coordinatorContract,
    consumerContract,
    coordinatorContractOracleSigner,

    state
  }
}

function verifyRequest(state, txReceipt) {
  expect(txReceipt.events.length).to.be.equal(1)
  const requestEvent = state.coordinatorContract.interface.parseLog(txReceipt.events[0])
  expect(requestEvent.name).to.be.equal('DataRequested')

  for (const arg of EVENT_ARGS) {
    expect(requestEvent.args[arg]).to.not.be.undefined
  }

  const { accId, requestId } = requestEvent.args
  return { accId, requestId }
}

function verifyRequestDirectPayment(state, txReceipt) {
  expect(txReceipt.events.length).to.be.equal(3)

  // TemporaryAccountCreated
  const temporaryAccountCreatedEvent = state.prepaymentContract.interface.parseLog(
    txReceipt.events[0]
  )
  expect(temporaryAccountCreatedEvent.name).to.be.equal('TemporaryAccountCreated')

  // DataRequested
  const requestEvent = state.coordinatorContract.interface.parseLog(txReceipt.events[1])
  expect(requestEvent.name).to.be.equal('DataRequested')

  // AccountBalanceIncreased
  const accountBalanceIncreasedEvent = state.prepaymentContract.interface.parseLog(
    txReceipt.events[2]
  )
  expect(accountBalanceIncreasedEvent.name).to.be.equal('AccountBalanceIncreased')

  for (const arg of EVENT_ARGS) {
    expect(requestEvent.args[arg]).to.not.be.undefined
  }

  const { accId, requestId } = requestEvent.args
  return { accId, requestId }
}

async function verifyFulfillment(
  state,
  txReceipt,
  accId,
  requestId,
  responseValue,
  responseFn,
  fulfillEventName
) {
  // AccountBalanceDecreased ////////////////////////////////////////////////////
  const prepaymentEvent = state.prepaymentContractConsumerSigner.interface.parseLog(
    txReceipt.events[0]
  )
  expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
  expect(prepaymentEvent.args.accId).to.be.equal(accId)

  // DataRequestFulfilled* //////////////////////////////////////////////////////
  const fulfillEvent = state.coordinatorContract.interface.parseLog(
    txReceipt.events[txReceipt.events.length - 1]
  )
  expect(fulfillEvent.name).to.be.equal(fulfillEventName)
  expect(fulfillEvent.args.requestId).to.be.equal(requestId)
  expect(await responseFn()).to.be.equal(responseValue)
}

async function requestAndFulfill(
  state,
  requestFn,
  fulfillFn,
  fulfillValue,
  getFulfillValueFn,
  fulfillEventName,
  isDirectPayment,
  numSubmission,
  type
) {
  const { rrOracle0, maxGasLimit, gasAfterPaymentCalculation, feeConfig, directFeeConfig } =
    await loadFixture(deployFixture)

  // Configure coordinator
  await state.coordinatorContract.registerOracle(rrOracle0)
  await state.coordinatorContract.setConfig(
    maxGasLimit,
    gasAfterPaymentCalculation,
    Object.values(feeConfig)
  )
  await state.coordinatorContract.setDirectPaymentConfig(directFeeConfig);

  // Request data /////////////////////////////////////////////////////////////
  const gasLimit = 500_000
  let requestReceipt
  if (isDirectPayment) {
    requestReceipt = await (
      await requestFn(maxGasLimit, numSubmission, {
        gasLimit,
        value: parseKlay(1)
      })
    ).wait()
  } else {
    const accId = await state.createAccount()
    await state.deposit('1')
    await state.addConsumer(state.consumerContract.address)
    requestReceipt = await (
      await requestFn(accId, maxGasLimit, numSubmission, {
        gasLimit
      })
    ).wait()
  }

  let _requestId
  let _accId
  if (isDirectPayment) {
    const { requestId, accId } = verifyRequestDirectPayment(state, requestReceipt)
    _requestId = requestId
    _accId = accId
  } else {
    const { requestId, accId } = verifyRequest(state, requestReceipt)
    _requestId = requestId
    _accId = accId
  }

  // Fulfill data //////////////////////////////////////////////////////////////
  const requestCommitment = {
    blockNum: requestReceipt.blockNumber,
    accId: _accId,
    callbackGasLimit: maxGasLimit,
    sender: state.consumerContract.address
  }

  let fulfillReceipt
  for (let i = 0; i < numSubmission; i++) {
    fulfillReceipt = await (
      await fulfillFn(_requestId, fulfillValue[i], requestCommitment, isDirectPayment)
    ).wait()
  }
  let responseValue = fulfillValue[0]
  if (type == 'Uint256' || type == 'int256') {
    responseValue = median(fulfillValue)
    console.log(type, responseValue)
  }
  if (type == 'bool') {
    let falseCount = 0
    let trueCount = 0
    for (let i = 0; i < fulfillValue.length; i++) {
      if (illValue[i]) trueCount++
      else falseCount++
    }
    responseValue = trueCount >= falseCount
  }

  await verifyFulfillment(
    state,
    fulfillReceipt,
    _accId,
    _requestId,
    responseValue,
    getFulfillValueFn,
    fulfillEventName
  )
}

function median(arr) {
  return Math.floor(medianFn(arr))
}

describe('Request-Response user contract', function () {
  it('requestData should revert with InsufficientPayment error', async function () {
    const { state, maxGasLimit } = await loadFixture(deployFixture)
    const accId = await state.createAccount()
    await state.addConsumer(state.consumerContract.address)
    const numSubmission = 1
    await expect(
      state.consumerContract.requestDataUint256(accId, maxGasLimit, numSubmission, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(state.coordinatorContract, 'InsufficientPayment')
  })

  it('Request & Fulfill Uint256', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 2
    await requestAndFulfill(
      state,
      state.consumerContract.requestDataUint256,
      state.coordinatorContractOracleSigner.fulfillDataRequestUint256,
      ['1', '2'],
      state.consumerContract.sResponseUint256,
      'DataRequestFulfilledUint256',
      false,
      numSubmission,
      'Uint256'
    )
  })

  it('Request & Fulfill Uint256 Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentUint256,
      state.coordinatorContractOracleSigner.fulfillDataRequestUint256,
      ['1', '2'],
      state.consumerContract.sResponseUint256,
      'DataRequestFulfilledUint256',
      true,
      numSubmission,
      'Uint256'
    )
  })

  it('Request & Fulfill Int256', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataInt256,
      state.coordinatorContractOracleSigner.fulfillDataRequestInt256,
      [10, 11],
      state.consumerContract.sResponseInt256,
      'DataRequestFulfilledInt256',
      false,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Int256 Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentInt256,
      state.coordinatorContractOracleSigner.fulfillDataRequestInt256,
      [10, 11],
      state.consumerContract.sResponseInt256,
      'DataRequestFulfilledInt256',
      true,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Bool', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 3

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBool,
      state.coordinatorContractOracleSigner.fulfillDataRequestBool,
      [true, false, true],
      state.consumerContract.sResponseBool,
      'DataRequestFulfilledBool',
      false,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill Bool Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 3

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBool,
      state.coordinatorContractOracleSigner.fulfillDataRequestBool,
      [false, true, false],
      state.consumerContract.sResponseBool,
      'DataRequestFulfilledBool',
      true,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill String', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataString,
      state.coordinatorContractOracleSigner.fulfillDataRequestString,
      ['hello'],
      state.consumerContract.sResponseString,
      'DataRequestFulfilledString',
      false,
      numSubmission
    )
  })

  it('Request & Fulfill String Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentString,
      state.coordinatorContractOracleSigner.fulfillDataRequestString,
      ['hello'],
      state.consumerContract.sResponseString,
      'DataRequestFulfilledString',
      true,
      numSubmission
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes32,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes32,
      [ethers.utils.formatBytes32String('hello')],
      state.consumerContract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      false,
      numSubmission
    )
  })

  it('Request & Fulfill Bytes32 Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBytes32,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes32,
      [ethers.utils.formatBytes32String('hello')],
      state.consumerContract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      true,
      numSubmission
    )
  })

  it('Request & Fulfill Bytes', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes,
      ['0x1234'],
      state.consumerContract.sResponseBytes,
      'DataRequestFulfilledBytes',
      false,
      numSubmission
    )
  })

  it('Request & Fulfill Bytes Direct Payment', async function () {
    const { state } = await loadFixture(deployFixture)
    const numSubmission = 1
    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBytes,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes,
      ['0x1234'],
      state.consumerContract.sResponseBytes,
      'DataRequestFulfilledBytes',
      true,
      numSubmission
    )
  })

  // TODO deregister oracle
  // TODO getters
  // TODO set direct payment config
  // TODO cancel request & pending request exist
  // TODO invalid consumer
  // TODO gas limit too big
  // TODO UnregisteredOracleFulfillment
  // TODO NoCorrespondingRequest
  // TODO IncorrectCommitment
})
