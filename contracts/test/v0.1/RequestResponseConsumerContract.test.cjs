const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const {
  deploy: deployCoordinator,
  parseDataRequestedTx,
  DATA_REQUEST_EVENT_ARGS,
  parseDataRequestFulfilledTx
} = require('./RequestResponseCoordinator.utils.cjs')
const { parseKlay } = require('./utils.cjs')
const { median, majorityVotingBool } = require('./utils.cjs')
const {
  deploy: deployPrepayment,
  createAccount,
  addConsumer,
  deposit,
  addCoordinator
} = require('./Prepayment.utils.cjs')

async function setupOracle(coordinator, oracles) {
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()
  await coordinator.setConfig(maxGasLimit, gasAfterPaymentCalculation, Object.values(feeConfig))
  for (const oracle of oracles) {
    await coordinator.registerOracle(oracle.address)
  }
}

async function createSigners() {
  let {
    deployer,
    consumer,
    consumer1,
    consumer2,
    account8: account,
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3
  } = await hre.getNamedAccounts()

  const account0 = await ethers.getSigner(deployer)
  const account1 = await ethers.getSigner(consumer)
  const account2 = await ethers.getSigner(consumer1)
  const account3 = await ethers.getSigner(consumer2)
  const account4 = await ethers.getSigner(account)
  const account5 = await ethers.getSigner(rrOracle0)
  const account6 = await ethers.getSigner(rrOracle1)
  const account7 = await ethers.getSigner(rrOracle2)
  const account8 = await ethers.getSigner(rrOracle3)

  return {
    account0,
    account1,
    account2,
    account3,
    account4,
    account5,
    account6,
    account7,
    account8
  }
}

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2: rrOracle0,
    account3: rrOracle1,
    account4: rrOracle2,
    account5: rrOracle3,
    account6: rrOracle4,
    account7: rrOracle5,
    account8: protocolFeeRecipient
  } = await createSigners()
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()

  // Prepayment
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  const prepayment = { contract: prepaymentContract, signer: deployerSigner }

  // RequestResponseCoordinator
  const coordinatorContract = await deployCoordinator(prepayment.contract.address, deployerSigner)
  const coordinator = { contract: coordinatorContract, signer: deployerSigner }
  await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

  // RequestResponseConsumerMock
  let consumerContract = await ethers.getContractFactory('RequestResponseConsumerMock', {
    signer: consumerSigner
  })
  consumerContract = await consumerContract.deploy(coordinatorContract.address)
  await consumerContract.deployed()
  const consumer = { contract: consumerContract, signer: consumerSigner }

  return {
    rrOracle0,
    rrOracle1,
    rrOracle2,
    rrOracle3,
    rrOracle4,
    rrOracle5,
    protocolFeeRecipient,

    prepayment,
    coordinator,
    consumer
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

function verifyRequestDirectPayment(prepaymentContract, coordinatorContract, txReceipt) {
  expect(txReceipt.events.length).to.be.equal(3)

  // TemporaryAccountCreated
  const temporaryAccountCreatedEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
  expect(temporaryAccountCreatedEvent.name).to.be.equal('TemporaryAccountCreated')

  // DataRequested
  const requestEvent = coordinatorContract.interface.parseLog(txReceipt.events[1])
  expect(requestEvent.name).to.be.equal('DataRequested')

  // AccountBalanceIncreased
  const accountBalanceIncreasedEvent = prepaymentContract.interface.parseLog(txReceipt.events[2])
  expect(accountBalanceIncreasedEvent.name).to.be.equal('AccountBalanceIncreased')

  for (const arg of DATA_REQUEST_EVENT_ARGS) {
    expect(requestEvent.args[arg]).to.not.be.undefined
  }

  const { accId, requestId } = requestEvent.args
  return { accId, requestId }
}

async function verifyFulfillment(
  prepayment,
  coordinator,
  txReceipt,
  accId,
  requestId,
  responseValue,
  responseFn,
  fulfillEventName
) {
  // AccountBalanceDecreased ////////////////////////////////////////////////////
  const prepaymentEvent = prepayment.contract.interface.parseLog(txReceipt.events[0])
  expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
  expect(prepaymentEvent.args.accId).to.be.equal(accId)

  // DataRequestFulfilled* //////////////////////////////////////////////////////
  const { requestId: eventRequestId } = parseDataRequestFulfilledTx(
    coordinator.contract,
    txReceipt,
    fulfillEventName
  )
  expect(await responseFn()).to.be.equal(responseValue)
}

async function requestAndFulfill(
  oracles,
  requestFn,
  fulfillFnName,
  fulfillValue,
  getFulfillValueFn,
  fulfillEventName,
  isDirectPayment,
  numSubmission,
  dataType
) {
  const { prepayment, coordinator, consumer } = await loadFixture(deploy)
  const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
  await setupOracle(coordinator.contract, oracles)

  // Request data /////////////////////////////////////////////////////////////
  const gasLimit = 500_000
  let requestReceipt
  if (isDirectPayment) {
    requestReceipt = await (
      await requestFn(callbackGasLimit, numSubmission, consumer.signer.address, {
        gasLimit,
        value: parseKlay(1)
      })
    ).wait()
  } else {
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    requestReceipt = await (
      await requestFn(accId, callbackGasLimit, numSubmission, {
        gasLimit
      })
    ).wait()
  }

  // Verify Request
  let _requestId
  let _accId
  if (isDirectPayment) {
    const { requestId, accId } = verifyRequestDirectPayment(
      prepayment.contract,
      coordinator.contract,
      requestReceipt
    )
    _requestId = requestId
    _accId = accId
  } else {
    const { requestId, accId } = parseDataRequestedTx(coordinator.contract, requestReceipt)
    _requestId = requestId
    _accId = accId
  }

  // Fulfill data //////////////////////////////////////////////////////////////
  const requestCommitment = {
    blockNum: requestReceipt.blockNumber,
    accId: _accId,
    callbackGasLimit,
    numSubmission,
    sender: consumer.contract.address
  }

  let fulfillReceipt
  for (let i = 0; i < numSubmission; i++) {
    fulfillReceipt = await (
      await coordinator.contract
        .connect(oracles[i])
        [fulfillFnName](_requestId, fulfillValue[i], requestCommitment, isDirectPayment)
    ).wait()
  }

  const responseValue = aggregateSubmissions(fulfillValue, dataType)

  // Verify Fulfillment
  await verifyFulfillment(
    prepayment,
    coordinator,
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
    const { prepayment, consumer, coordinator } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    const numSubmission = 1
    await expect(
      consumer.contract.requestDataUint128(accId, callbackGasLimit, numSubmission, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(coordinator.contract, 'InsufficientPayment')
  })

  it('Request & Fulfill Uint128', async function () {
    const { consumer, rrOracle0, rrOracle1 } = await loadFixture(deploy)
    const numSubmission = 2
    await requestAndFulfill(
      [rrOracle0, rrOracle1],
      consumer.contract.requestDataUint128,
      'fulfillDataRequestUint128',
      [1, 2],
      consumer.contract.sResponseUint128,
      'DataRequestFulfilledUint128',
      false,
      numSubmission,
      'Uint128'
    )
  })

  it('Request & Fulfill Uint128 Direct Payment', async function () {
    const { consumer, rrOracle0, rrOracle1 } = await loadFixture(deploy)
    const numSubmission = 2

    await requestAndFulfill(
      [rrOracle0, rrOracle1],
      consumer.contract.requestDataDirectPaymentUint128,
      'fulfillDataRequestUint128',
      [1, 2],
      consumer.contract.sResponseUint128,
      'DataRequestFulfilledUint128',
      true,
      numSubmission,
      'Uint128'
    )
  })

  it('Request & Fulfill Int256', async function () {
    const { consumer, rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4 } = await loadFixture(
      deploy
    )
    const numSubmission = 2

    await requestAndFulfill(
      [rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4],
      consumer.contract.requestDataInt256,
      'fulfillDataRequestInt256',
      [10, 11],
      consumer.contract.sResponseInt256,
      'DataRequestFulfilledInt256',
      false,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Int256 Direct Payment', async function () {
    const { consumer, rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4 } = await loadFixture(
      deploy
    )
    const numSubmission = 2

    await requestAndFulfill(
      [rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4],
      consumer.contract.requestDataDirectPaymentInt256,
      'fulfillDataRequestInt256',
      [10, 11],
      consumer.contract.sResponseInt256,
      'DataRequestFulfilledInt256',
      true,
      numSubmission,
      'Int256'
    )
  })

  it('Request & Fulfill Bool', async function () {
    const { consumer, rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4, rrOracle5 } =
      await loadFixture(deploy)
    const numSubmission = 3

    await requestAndFulfill(
      [rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4, rrOracle5],
      consumer.contract.requestDataBool,
      'fulfillDataRequestBool',
      [true, false, true],
      consumer.contract.sResponseBool,
      'DataRequestFulfilledBool',
      false,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill Bool Direct Payment', async function () {
    const { consumer, rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4, rrOracle5 } =
      await loadFixture(deploy)
    const numSubmission = 3

    await requestAndFulfill(
      [rrOracle0, rrOracle1, rrOracle2, rrOracle3, rrOracle4, rrOracle5],
      consumer.contract.requestDataDirectPaymentBool,
      'fulfillDataRequestBool',
      [false, true, false],
      consumer.contract.sResponseBool,
      'DataRequestFulfilledBool',
      true,
      numSubmission,
      'Bool'
    )
  })

  it('Request & Fulfill String', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataString,
      'fulfillDataRequestString',
      ['hello'],
      consumer.contract.sResponseString,
      'DataRequestFulfilledString',
      false,
      numSubmission,
      'String'
    )
  })

  it('Request & Fulfill String Direct Payment', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataDirectPaymentString,
      'fulfillDataRequestString',
      ['hello'],
      consumer.contract.sResponseString,
      'DataRequestFulfilledString',
      true,
      numSubmission,
      'String'
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataBytes32,
      'fulfillDataRequestBytes32',
      [ethers.utils.formatBytes32String('hello')],
      consumer.contract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      false,
      numSubmission,
      'Bytes32'
    )
  })

  it('Request & Fulfill Bytes32 Direct Payment', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataDirectPaymentBytes32,
      'fulfillDataRequestBytes32',
      [ethers.utils.formatBytes32String('hello')],
      consumer.contract.sResponseBytes32,
      'DataRequestFulfilledBytes32',
      true,
      numSubmission,
      'Bytes32'
    )
  })

  it('Request & Fulfill Bytes', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1

    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataBytes,
      'fulfillDataRequestBytes',
      ['0x1234'],
      consumer.contract.sResponseBytes,
      'DataRequestFulfilledBytes',
      false,
      numSubmission,
      'Bytes'
    )
  })

  it('Request & Fulfill Bytes Direct Payment', async function () {
    const { consumer, rrOracle0 } = await loadFixture(deploy)
    const numSubmission = 1
    await requestAndFulfill(
      [rrOracle0],
      consumer.contract.requestDataDirectPaymentBytes,
      'fulfillDataRequestBytes',
      ['0x1234'],
      consumer.contract.sResponseBytes,
      'DataRequestFulfilledBytes',
      true,
      numSubmission,
      'Bytes'
    )
  })

  it('cancel request for [regular] account', async function () {
    const { prepayment, consumer, coordinator, rrOracle0 } = await loadFixture(deploy)
    await setupOracle(coordinator.contract, [rrOracle0])
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 1

    // Request data /////////////////////////////////////////////////////////////
    const requestReceipt = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()
    const { requestId } = parseDataRequestedTx(coordinator.contract, requestReceipt)

    // Cancel Request ///////////////////////////////////////////////////////////
    const txCancelRequest = await (await consumer.contract.cancelRequest(requestId)).wait()

    const dataRequestCancelledEvent = coordinator.contract.interface.parseLog(
      txCancelRequest.events[0]
    )
    expect(dataRequestCancelledEvent.name).to.be.equal('RequestCanceled')

    const { requestId: cRequestId } = dataRequestCancelledEvent.args
    expect(requestId).to.be.equal(cRequestId)
  })

  it('increase nonce by every request with [regular] account', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()

    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 1

    // Before first request
    const nonce1 = await prepayment.contract.getNonce(accId, consumer.contract.address)
    expect(nonce1).to.be.equal(1)
    await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)

    // After first request
    const nonce2 = await prepayment.contract.getNonce(accId, consumer.contract.address)
    expect(nonce2).to.be.equal(2)
    await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)

    // After second request
    const nonce3 = await prepayment.contract.getNonce(accId, consumer.contract.address)
    expect(nonce3).to.be.equal(3)
  })

  it('increase reqCount by every request with [regular] account', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 1

    // Before first request, `reqCount` should be 0
    const reqCountBeforeRequest = await prepayment.contract.getReqCount(accId)
    expect(reqCountBeforeRequest).to.be.equal(0)
    const requestDataTx = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()

    const { requestId, sender, blockNumber, isDirectPayment } = parseDataRequestedTx(
      coordinator.contract,
      requestDataTx
    )

    // The `reqCount` after the request does not change. It gets
    // updated during `chargeFee` call inside of `Account` contract.
    const reqCountAfterRequest = await prepayment.contract.getReqCount(accId)
    expect(reqCountAfterRequest).to.be.equal(0)

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }

    await coordinator.contract
      .connect(rrOracle0)
      .fulfillDataRequestInt256(requestId, 123, requestCommitment, isDirectPayment)

    // The value of `reqCount` should increase
    const reqCountAfterFulfillment = await prepayment.contract.getReqCount(accId)
    expect(reqCountAfterFulfillment).to.be.equal(1)
  })

  it('TooManyOracles', async function () {
    const { coordinator } = await loadFixture(deploy)
    const MAX_ORACLES = await coordinator.contract.MAX_ORACLES()

    for (let i = 0; i < MAX_ORACLES; ++i) {
      const { address: oracle } = ethers.Wallet.createRandom()
      await coordinator.contract.registerOracle(oracle)
    }

    const { address: oracle } = ethers.Wallet.createRandom()
    await expect(coordinator.contract.registerOracle(oracle)).to.be.revertedWithCustomError(
      coordinator.contract,
      'TooManyOracles'
    )
  })

  it('PendingRequestExists', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request
    const numSubmission = 1
    const tx = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()
    const { requestId } = parseDataRequestedTx(coordinator.contract, tx)

    // nonce 1 represents a valid account
    // nonce 2 represents the first request
    const nonce = 2
    const pendingRequestExists = await coordinator.contract
      .connect(consumer.signer)
      .pendingRequestExists(consumer.contract.address, accId, nonce)
    expect(pendingRequestExists).to.be.equal(true)
  })

  it('InsufficientPayment', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Request
    const numSubmission = 1
    await expect(
      consumer.contract.requestDataDirectPaymentInt256(
        callbackGasLimit,
        numSubmission,
        consumer.signer.address,
        {
          value: 0
        }
      )
    ).to.be.revertedWithCustomError(coordinator.contract, 'InsufficientPayment')
  })

  it('InvalidConsumer', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    // Did not assign consumer to account!

    // Request
    const numSubmission = 1
    await expect(
      consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).to.be.revertedWithCustomError(coordinator.contract, 'InvalidConsumer')
  })

  it('GasLimitTooBig', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)

    // Request
    const numSubmission = 1
    const callbackGasLimit = maxGasLimit + 1
    await expect(
      consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).to.be.revertedWithCustomError(coordinator.contract, 'GasLimitTooBig')
  })

  it('UnregisteredOracleFulfillment', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()
    await coordinator.contract.setConfig(
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 1
    const callbackGasLimit = maxGasLimit
    const requestTx = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()

    const { requestId, sender, blockNumber, isDirectPayment } = parseDataRequestedTx(
      coordinator.contract,
      requestTx
    )

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }

    const response = 123
    await expect(
      coordinator.contract
        .connect(rrOracle0)
        .fulfillDataRequestInt256(requestId, response, requestCommitment, isDirectPayment)
    ).to.be.revertedWithCustomError(coordinator.contract, 'UnregisteredOracleFulfillment')
  })

  it('OracleAlreadySubmitted', async function () {
    const { prepayment, coordinator, consumer, rrOracle0, rrOracle1, rrOracle2, rrOracle3 } =
      await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0, rrOracle1, rrOracle2, rrOracle3])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 2
    const requestTx = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()

    const { requestId, sender, blockNumber, isDirectPayment } = parseDataRequestedTx(
      coordinator.contract,
      requestTx
    )

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }

    const response = 123
    await coordinator.contract
      .connect(rrOracle0)
      .fulfillDataRequestInt256(requestId, response, requestCommitment, isDirectPayment)

    await expect(
      coordinator.contract
        .connect(rrOracle0)
        .fulfillDataRequestInt256(requestId, response, requestCommitment, isDirectPayment)
    ).to.be.revertedWithCustomError(coordinator.contract, 'OracleAlreadySubmitted')
  })

  it('NoCorrespondingRequest', async function () {
    const { prepayment, coordinator, consumer, rrOracle0 } = await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    await setupOracle(coordinator.contract, [rrOracle0])

    // Prepare account
    const { accId } = await createAccount(prepayment.contract, consumer.signer)
    await addConsumer(prepayment.contract, consumer.signer, accId, consumer.contract.address)
    await deposit(prepayment.contract, consumer.signer, accId, parseKlay(1))

    // Request configuration
    const numSubmission = 1
    const requestTx = await (
      await consumer.contract.requestDataInt256(accId, callbackGasLimit, numSubmission)
    ).wait()

    const { sender, blockNumber, isDirectPayment } = parseDataRequestedTx(
      coordinator.contract,
      requestTx
    )

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }

    const response = 123
    const wrongRequestId = 111
    await expect(
      coordinator.contract
        .connect(rrOracle0)
        .fulfillDataRequestInt256(wrongRequestId, response, requestCommitment, isDirectPayment)
    ).to.be.revertedWithCustomError(coordinator.contract, 'NoCorrespondingRequest')
  })

  // TODO getters
  // TODO IncorrectCommitment
})
