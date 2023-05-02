const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { State } = require('./State.utils.cjs')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const {
  parseDataRequestedTx,
  DATA_REQUEST_EVENT_ARGS,
  parseDataRequestFulfilledTx
} = require('./RequestResponseCoordinator.utils.cjs')
const { parseKlay } = require('./utils.cjs')
const { median, majorityVotingBool } = require('./utils.cjs')

async function setupOracle(coordinator, oracles) {
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()

  for (const oracle of oracles) {
    await coordinator.registerOracle(oracle)
  }
  await coordinator.setConfig(maxGasLimit, gasAfterPaymentCalculation, Object.values(feeConfig))
}

async function createSigners() {
  let { rrOracle0 } = await hre.getNamedAccounts()

  const rrOracle0Signer = await ethers.getSigner(rrOracle0)

  return {
    rrOracle0Signer
  }
}

async function deploy() {
  const {
    deployer,
    consumer,
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3,
    rrOracle4,
    rrOracle5,
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

  // Oracles ////////////////////////////////////////////////////////////////////
  const coordinatorContractOracleSigner0 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle0
  )

  const coordinatorContractOracleSigner1 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle1
  )

  const coordinatorContractOracleSigner2 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle2
  )

  const coordinatorContractOracleSigner3 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle3
  )

  const coordinatorContractOracleSigner4 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle4
  )

  const coordinatorContractOracleSigner5 = await ethers.getContractAt(
    'RequestResponseCoordinator',
    coordinatorContract.address,
    rrOracle5
  )

  // State controller ///////////////////////////////////////////////////////////
  const state = new State(consumer, prepaymentContract, consumerContract, coordinatorContract, [
    coordinatorContractOracleSigner0,
    coordinatorContractOracleSigner1,
    coordinatorContractOracleSigner2,
    coordinatorContractOracleSigner3,
    coordinatorContractOracleSigner4,
    coordinatorContractOracleSigner5
  ])
  await state.initialize('RequestResponseConsumerMock')
  await state.addCoordinator(coordinatorContract.address)

  return {
    deployer,
    consumer,
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3,
    rrOracle4,
    rrOracle5,

    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
    prepaymentContract,
    coordinatorContract,
    consumerContract,

    state
  }
}

function aggregateSubmissions(arr, dataType) {
  expect(arr.length).to.be.greaterThan(0)

  switch (dataType.toLowerCase()) {
    case 'uint256':
    case 'int256':
      return median(arr)
    case 'bool':
      return majorityVotingBool(arr)
    default:
      return arr[0]
  }
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

  for (const arg of DATA_REQUEST_EVENT_ARGS) {
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
  const { requestId: eventRequestId } = parseDataRequestFulfilledTx(
    state.coordinatorContract,
    txReceipt,
    fulfillEventName
  )
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
  dataType
) {
  const {
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3,
    rrOracle4,
    rrOracle5,
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig,
    consumer
  } = await loadFixture(deploy)

  await setupOracle(state.coordinatorContract, [
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3,
    rrOracle4,
    rrOracle5
  ])

  // Request data /////////////////////////////////////////////////////////////
  const gasLimit = 500_000
  let requestReceipt
  if (isDirectPayment) {
    requestReceipt = await (
      await requestFn(maxGasLimit, numSubmission, consumer, {
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

  // Verify Request
  let _requestId
  let _accId
  if (isDirectPayment) {
    const { requestId, accId } = verifyRequestDirectPayment(state, requestReceipt)
    _requestId = requestId
    _accId = accId
  } else {
    const { requestId, accId } = parseDataRequestedTx(state.coordinatorContract, requestReceipt)
    _requestId = requestId
    _accId = accId
  }

  // Fulfill data //////////////////////////////////////////////////////////////
  const requestCommitment = {
    blockNum: requestReceipt.blockNumber,
    accId: _accId,
    callbackGasLimit: maxGasLimit,
    numSubmission,
    sender: state.consumerContract.address
  }

  let fulfillReceipt
  for (let i = 0; i < numSubmission; i++) {
    fulfillReceipt = await (
      await fulfillFn[i](_requestId, fulfillValue[i], requestCommitment, isDirectPayment)
    ).wait()

    if (numSubmission > 1 && i < numSubmission - 1) {
      await expect(
        fulfillFn[i](_requestId, fulfillValue[i], requestCommitment, isDirectPayment)
      ).to.be.revertedWithCustomError(state.coordinatorContract, 'OracleAlreadySubmitted')
    }
  }

  const responseValue = aggregateSubmissions(fulfillValue, dataType)

  // Verify Fulfillment
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

describe('Request-Response user contract', function () {
  it('requestData should revert with InsufficientPayment error', async function () {
    const { state, maxGasLimit } = await loadFixture(deploy)
    const accId = await state.createAccount()
    await state.addConsumer(state.consumerContract.address)
    const numSubmission = 1
    await expect(
      state.consumerContract.requestDataUint128(accId, maxGasLimit, numSubmission, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(state.coordinatorContract, 'InsufficientPayment')
  })

  it('Request & Fulfill Uint128', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 2
    await requestAndFulfill(
      state,
      state.consumerContract.requestDataUint128,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestUint128,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestUint128
      ],
      [1, 2],
      state.consumerContract.sResponseUint128,
      'DataRequestFulfilledUint128',
      false,
      numSubmission,
      'Uint128'
    )
  })

  it('Request & Fulfill Uint128 Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentUint128,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestUint128,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestUint128
      ],
      [1, 2],
      state.consumerContract.sResponseUint128,
      'DataRequestFulfilledUint128',
      true,
      numSubmission,
      'Uint128'
    )
  })

  it('Request & Fulfill Int256', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataInt256,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestInt256,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestInt256
      ],
      [10, 11],
      state.consumerContract.sResponseInt256,
      'DataRequestFulfilledInt256',
      false,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Int256 Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 2

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentInt256,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestInt256,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestInt256
      ],
      [10, 11],
      state.consumerContract.sResponseInt256,
      'DataRequestFulfilledInt256',
      true,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Bool', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 3

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBool,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestBool,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestBool,
        state.coordinatorContractOracleSigners[2].fulfillDataRequestBool
      ],
      [true, false, true],
      state.consumerContract.sResponseBool,
      'DataRequestFulfilledBool',
      false,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill Bool Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 3

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBool,
      [
        state.coordinatorContractOracleSigners[0].fulfillDataRequestBool,
        state.coordinatorContractOracleSigners[1].fulfillDataRequestBool,
        state.coordinatorContractOracleSigners[2].fulfillDataRequestBool
      ],
      [false, true, false],
      state.consumerContract.sResponseBool,
      'DataRequestFulfilledBool',
      true,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill String', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataString,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestString],
      ['hello'],
      state.consumerContract.sResponseString,
      'DataRequestFulfilledString',
      false,
      numSubmission,
      'String'
    )
  })

  it('Request & Fulfill String Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentString,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestString],
      ['hello'],
      state.consumerContract.sResponseString,
      'DataRequestFulfilledString',
      true,
      numSubmission,
      'String'
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes32,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestBytes32],
      [ethers.utils.formatBytes32String('hello')],
      state.consumerContract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      false,
      numSubmission,
      'Bytes32'
    )
  })

  it('Request & Fulfill Bytes32 Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBytes32,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestBytes32],
      [ethers.utils.formatBytes32String('hello')],
      state.consumerContract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      true,
      numSubmission,
      'Bytes32'
    )
  })

  it('Request & Fulfill Bytes', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      state,
      state.consumerContract.requestDataBytes,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestBytes],
      ['0x1234'],
      state.consumerContract.sResponseBytes,
      'DataRequestFulfilledBytes',
      false,
      numSubmission,
      'Bytes'
    )
  })

  it('Request & Fulfill Bytes Direct Payment', async function () {
    const { state } = await loadFixture(deploy)
    const numSubmission = 1
    await requestAndFulfill(
      state,
      state.consumerContract.requestDataDirectPaymentBytes,
      [state.coordinatorContractOracleSigners[0].fulfillDataRequestBytes],
      ['0x1234'],
      state.consumerContract.sResponseBytes,
      'DataRequestFulfilledBytes',
      true,
      numSubmission,
      'Bytes'
    )
  })

  it('cancel request for [regular] account', async function () {
    const { state, rrOracle0, maxGasLimit: callbackGasLimit } = await loadFixture(deploy)
    await setupOracle(state.coordinatorContract, [rrOracle0])

    // Prepare account
    const accId = await state.createAccount()
    await state.deposit('1')
    await state.addConsumer(state.consumerContract.address)

    // Request configuration
    const numSubmission = 1

    // Request data /////////////////////////////////////////////////////////////
    const requestReceipt = await (
      await state.consumerContract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()
    const { requestId } = parseDataRequestedTx(state.coordinatorContract, requestReceipt)

    // Cancel Request ///////////////////////////////////////////////////////////
    const txCancelRequest = await (await state.consumerContract.cancelRequest(requestId)).wait()

    const dataRequestCancelledEvent = state.coordinatorContract.interface.parseLog(
      txCancelRequest.events[0]
    )
    expect(dataRequestCancelledEvent.name).to.be.equal('RequestCanceled')

    const { requestId: cRequestId } = dataRequestCancelledEvent.args
    expect(requestId).to.be.equal(cRequestId)
  })

  it('increase nonce by every request with [regular] account', async function () {
    const { state, rrOracle0, maxGasLimit: callbackGasLimit } = await loadFixture(deploy)

    await setupOracle(state.coordinatorContract, [rrOracle0])

    // Prepare account
    const accId = await state.createAccount()
    await state.deposit('1')
    await state.addConsumer(state.consumerContract.address)

    // Request configuration
    const numSubmission = 1

    // Before first request
    const nonce1 = await state.prepaymentContract.getNonce(accId, state.consumerContract.address)
    expect(nonce1).to.be.equal(1)
    await state.consumerContract.requestDataInt256(accId, callbackGasLimit, numSubmission)

    // After first request
    const nonce2 = await state.prepaymentContract.getNonce(accId, state.consumerContract.address)
    expect(nonce2).to.be.equal(2)
    await state.consumerContract.requestDataInt256(accId, callbackGasLimit, numSubmission)

    // After second request
    const nonce3 = await state.prepaymentContract.getNonce(accId, state.consumerContract.address)
    expect(nonce3).to.be.equal(3)
  })

  it('increase reqCount by every request with [regular] account', async function () {
    const { state, rrOracle0, maxGasLimit: callbackGasLimit } = await loadFixture(deploy)
    const { rrOracle0Signer } = await createSigners()
    await setupOracle(state.coordinatorContract, [rrOracle0])

    // Prepare account
    const accId = await state.createAccount()
    await state.deposit('1')
    await state.addConsumer(state.consumerContract.address)

    // Request configuration
    const numSubmission = 1

    // Before first request, `reqCount` should be 0
    const reqCountBeforeRequest = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountBeforeRequest).to.be.equal(0)
    const requestDataTx = await (
      await state.consumerContract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()

    const { requestId, sender, blockNumber, isDirectPayment } = parseDataRequestedTx(
      state.coordinatorContract,
      requestDataTx
    )

    // The `reqCount` after the request does not change. It gets
    // updated during `chargeFee` call inside of `Account` contract.
    const reqCountAfterRequest = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountAfterRequest).to.be.equal(0)

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }

    await state.coordinatorContract
      .connect(rrOracle0Signer)
      .fulfillDataRequestInt256(requestId, 123, requestCommitment, isDirectPayment)

    // The value of `reqCount` should increase
    const reqCountAfterFulfillment = await state.prepaymentContract.getReqCount(accId)
    expect(reqCountAfterFulfillment).to.be.equal(1)
  })

  // TODO getters
  // TODO pending request exist
  // TODO invalid consumer
  // TODO gas limit too big
  // TODO UnregisteredOracleFulfillment
  // TODO NoCorrespondingRequest
  // TODO IncorrectCommitment
})
