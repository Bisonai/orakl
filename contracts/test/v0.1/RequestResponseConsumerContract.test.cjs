const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { State } = require('./State.utils.cjs')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const { parseKlay } = require('./utils.cjs')

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
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()

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

    prepaymentContract,
    coordinatorContract,
    consumerContract,
    coordinatorContractOracleSigner,

    state
  }
}

function verifyRequest(S, txReceipt) {
  expect(txReceipt.events.length).to.be.equal(1)
  const requestEvent = S.coordinatorContract.interface.parseLog(txReceipt.events[0])
  expect(requestEvent.name).to.be.equal('DataRequested')

  for (const arg of EVENT_ARGS) {
    expect(requestEvent.args[arg]).to.not.be.undefined
  }

  return requestEvent.args.requestId
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
  const fulfillEvent = state.coordinatorContract.interface.parseLog(txReceipt.events[1])
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
  fulfillEventName
) {
  const { rrOracle0, maxGasLimit, gasAfterPaymentCalculation, feeConfig } = await loadFixture(
    deployFixture
  )

  // Configure coordinator
  await state.coordinatorContract.registerOracle(rrOracle0)
  await state.coordinatorContract.setConfig(
    maxGasLimit,
    gasAfterPaymentCalculation,
    Object.values(feeConfig)
  )

  // Define state
  const accId = await state.createAccount()
  await state.addConsumer(state.consumerContract.address)
  await state.deposit('1')

  // Request data /////////////////////////////////////////////////////////////
  const requestReceipt = await (
    await requestFn(accId, maxGasLimit, {
      gasLimit: 500_000
    })
  ).wait()

  const requestId = verifyRequest(state, requestReceipt)

  // Fulfill data //////////////////////////////////////////////////////////////
  const requestCommitment = {
    blockNum: requestReceipt.blockNumber,
    accId,
    callbackGasLimit: maxGasLimit,
    sender: state.consumerContract.address
  }

  const isDirectPayment = false
  const fulfillReceipt = await (
    await fulfillFn(requestId, fulfillValue, requestCommitment, isDirectPayment)
  ).wait()

  await verifyFulfillment(
    state,
    fulfillReceipt,
    accId,
    requestId,
    fulfillValue,
    getFulfillValueFn,
    fulfillEventName
  )
}

describe('Request-Response user contract', function () {
  it('requestData should revert with InsufficientPayment error', async function () {
    const { state, maxGasLimit } = await loadFixture(deployFixture)

    const accId = await state.createAccount()
    await state.addConsumer(state.consumerContract.address)
    await expect(
      state.consumerContract.requestDataUint256(accId, maxGasLimit, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(state.coordinatorContract, 'InsufficientPayment')
  })

  it('Request & Fulfill Uint256', async function () {
    const { state } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataUint256,
      state.coordinatorContractOracleSigner.fulfillDataRequestUint256,
      123,
      state.consumerContract.sResponseUint256,
      'DataRequestFulfilledUint256'
    )
  })

  it('Request & Fulfill Int256', async function () {
    const { state } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataInt256,
      state.coordinatorContractOracleSigner.fulfillDataRequestInt256,
      -123,
      state.consumerContract.sResponseInt256,
      'DataRequestFulfilledInt256'
    )
  })

  it('Request & Fulfill bool', async function () {
    const {
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      coordinatorContractOracleSigner,
      state
    } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBool,
      state.coordinatorContractOracleSigner.fulfillDataRequestBool,
      true,
      state.consumerContract.sResponseBool,
      'DataRequestFulfilledBool'
    )
  })

  it('Request & Fulfill string', async function () {
    const { state } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataString,
      state.coordinatorContractOracleSigner.fulfillDataRequestString,
      'hello',
      state.consumerContract.sResponseString,
      'DataRequestFulfilledString'
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    const { state } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes32,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes32,
      ethers.utils.formatBytes32String('hello'),
      state.consumerContract.sResponseBytes32,
      'DataRequestFulfilledBytes32'
    )
  })

  it('Request & Fulfill Bytes', async function () {
    const { state } = await loadFixture(deployFixture)

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes,
      state.coordinatorContractOracleSigner.fulfillDataRequestBytes,
      '0x1234',
      state.consumerContract.sResponseBytes,
      'DataRequestFulfilledBytes'
    )
  })
})
