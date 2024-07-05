const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { vrfConfig } = require('../vrf/VRFCoordinator.config.cjs')
const { parseKlay, createSigners, median, majorityVotingBool } = require('../utils.cjs')
const {
  setupOracle,
  generateVrf,
  deploy: deployVrfCoordinator,
} = require('../vrf/VRFCoordinator.utils.cjs')
const { deploy: deployPrepayment, addCoordinator } = require('./Prepayment.utils.cjs')

const { propose, confirm, createAccount, addConsumer } = require('./Registry.utils.cjs')

const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const {
  deploy: deployCoordinator,
  setupOracle: setupRROracle,
} = require('./RequestResponseCoordinator.utils.cjs')

const SINGLE_WORD = 1
async function fulfillRandomWords(
  coordinator,
  registeredOracleSigner,
  notRegisteredOracleSigner,
  pi,
  rc,
  isDirectPayment,
) {
  // Random word request cannot be fulfilled by an unregistered oracle
  await expect(
    coordinator.connect(notRegisteredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment),
  ).to.be.revertedWithCustomError(coordinator, 'NoSuchProvingKey')

  // Registered oracle can submit data back to chain
  const tx = await (
    await coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).wait()

  // However even registered oracle cannot fulfill the request more than once
  await expect(
    coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment),
  ).to.be.revertedWithCustomError(coordinator, 'NoCorrespondingRequest')

  return tx
}

function validateRandomWordsRequestedEvent(
  tx,
  coordinatorContract,
  keyHash,
  accId,
  maxGasLimit,
  numWords,
  sender,
  isDirectPayment,
) {
  let eventIndex = expect(tx.events.length).to.be.equal(5)
  eventIndex = 2

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
    isDirectPayment: eIsDirectPayment,
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

function aggregateSubmissions(arr, dataType) {
  expect(arr.length).to.be.greaterThan(0)

  switch (dataType.toLowerCase()) {
    case 'uint128':
    case 'int256':
      return median(arr)
    case 'bool':
      return majorityVotingBool(arr)
    default:
      return arr[0]
  }
}

async function verifyFulfillment(prepayment, endpoint, txReceipt, accId, responseValue) {
  // AccountBalanceDecreased ////////////////////////////////////////////////////
  const prepaymentEvent = prepayment.contract.interface.parseLog(txReceipt.events[1])
  expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
  expect(prepaymentEvent.args.accId).to.be.equal(accId)

  // DataRequestFulfilled * //////////////////////////////////////////////////////
  const endpointEvent = endpoint.contract.interface.parseLog(txReceipt.events[0])
  const {
    requestId,
    l2RequestId,
    sender,
    callbackGasLimit,
    jobId,
    responseUint128,
    responseInt256,
    responseBool,
    responseString,
    responseBytes32,
    responseBytes,
  } = endpointEvent.args
  switch (jobId) {
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('uint128')):
      expect(responseUint128).to.be.equal(responseValue)
      break
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('int256')):
      expect(responseInt256).to.be.equal(responseValue)
      break
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('bool')):
      expect(responseBool).to.be.equal(responseValue)
      break
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('string')):
      expect(responseString).to.be.equal(responseValue)
      break
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('bytes32')):
      expect(responseBytes32).to.be.equal(responseValue)
      break
    case ethers.utils.keccak256(ethers.utils.toUtf8Bytes('bytes')):
      expect(responseBytes).to.be.equal(responseValue)
      break
  }
}

async function requestAndFulfill(
  requestFnName,
  fulfillFnName,
  fulfillValue,
  fulfillEventName,
  dataType,
  numSubmission,
) {
  const {
    l1Endpoint,
    rRCoordinator,
    consumer,
    prepayment,
    registry,
    registrAccount,
    account2: rrOracle1,
    account3: rrOracle2,
  } = await loadFixture(deploy)
  const oracles = [rrOracle1, rrOracle2]
  const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
  // Prepare coordinator
  for (let i = 0; i < oracles.length; i++) {
    await setupRROracle(rRCoordinator.contract, oracles[i].address)
  }

  //send balance for endpoint contract
  //deposit
  await registry.contract.deposit(registrAccount, { value: parseKlay('1') })
  const accBalance = await registry.contract.getBalance(registrAccount)
  expect(accBalance).to.be.equal(parseKlay('1'))

  // Request random words

  const l2RequestId = 1
  const txRequestData = await (
    await consumer.contract[requestFnName](
      registrAccount,
      callbackGasLimit,
      numSubmission,
      l2RequestId,
    )
  ).wait()

  expect(txRequestData.events.length).to.be.equal(5)
  const requestEvent = l1Endpoint.contract.interface.parseLog(txRequestData.events[4])
  expect(requestEvent.name).to.be.equal('DataRequested')

  // Fulfill data //////////////////////////////////////////////////////////////
  const requestEventData = rRCoordinator.contract.interface.parseLog(txRequestData.events[2])
  const { jobId, accId, sender, requestId } = requestEventData.args
  const isDirectPayment = true
  const requestCommitment = {
    blockNum: txRequestData.blockNumber,
    accId,
    callbackGasLimit,
    numSubmission,
    sender,
    isDirectPayment,
    jobId,
  }

  let fulfillReceipt
  for (let i = 0; i < numSubmission; i++) {
    fulfillReceipt = await (
      await rRCoordinator.contract
        .connect(oracles[i])
        [fulfillFnName](requestId, fulfillValue[i], requestCommitment)
    ).wait()
  }

  const responseValue = aggregateSubmissions(fulfillValue, dataType)

  // Verify Fulfillment
  await verifyFulfillment(prepayment, l1Endpoint, fulfillReceipt, accId, responseValue)
}

async function deploy() {
  const {
    account0: deployerSigner,
    account2,
    account3,
    account4: protocolFeeRecipient,
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner,
  }

  // VRFCoordinator

  const coordinatorContract = await deployVrfCoordinator(prepaymentContract.address, deployerSigner)
  expect(await coordinatorContract.typeAndVersion()).to.be.equal('VRFCoordinator v0.1')
  const coordinator = {
    contract: coordinatorContract,
    signer: deployerSigner,
  }
  await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

  const rRCoordinatorContract = await deployCoordinator(prepayment.contract.address, deployerSigner)
  const rRCoordinator = { contract: rRCoordinatorContract, signer: deployerSigner }
  await addCoordinator(prepayment.contract, prepayment.signer, rRCoordinator.contract.address)

  // registry

  let registryContract = await ethers.getContractFactory('Registry', {
    signer: deployerSigner,
  })
  registryContract = await registryContract.deploy()
  await registryContract.deployed()
  //setup registry

  const fee = parseKlay(1)
  const pChainID = '100001'
  const jsonRpc = 'https://123'
  const L2Endpoint = account2.address
  const { chainID } = await propose(
    registryContract,
    deployerSigner,
    pChainID,
    jsonRpc,
    L2Endpoint,
    fee,
  )
  await confirm(registryContract, deployerSigner, chainID)
  const { accId: rAccId } = await createAccount(registryContract, deployerSigner, chainID)
  //add consumer
  await addConsumer(registryContract, deployerSigner, rAccId, deployerSigner.address)

  let endpointContract = await ethers.getContractFactory('L1Endpoint', {
    signer: deployerSigner,
  })
  endpointContract = await endpointContract.deploy(
    registryContract.address,
    coordinatorContract.address,
    rRCoordinatorContract.address,
  )
  await endpointContract.deployed()
  await endpointContract.addOracle(deployerSigner.address)

  //add endpoint for registry
  await registryContract.setL1Endpoint(endpointContract.address)

  const l1Endpoint = {
    contract: endpointContract,
    signer: deployerSigner,
  }

  const registry = {
    contract: registryContract,
    signer: deployerSigner,
  }

  // consumer
  let consumerMock = await ethers.getContractFactory('L1EndpointConsumerMock', {
    signer: deployerSigner,
  })
  consumerMock = await consumerMock.deploy(endpointContract.address)
  await consumerMock.deployed()

  const consumer = {
    contract: consumerMock,
    signer: deployerSigner,
  }

  await addConsumer(registryContract, deployerSigner, rAccId, consumerMock.address)
  await endpointContract.addOracle(consumerMock.address)

  return {
    prepayment,
    coordinator,
    rRCoordinator,
    l1Endpoint,
    consumer,
    registry,
    account2,
    account3,
    registrAccount: rAccId,
  }
}

describe('L1Endpoint', function () {
  it('requestRandomWords', async function () {
    const {
      l1Endpoint,
      coordinator,
      registry,
      account2: oracle,
      account3: unregisteredOracle,
      registrAccount,
    } = await loadFixture(deploy)

    const { maxGasLimit: callbackGasLimit, keyHash } = vrfConfig()

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)

    //send balance for endpoint contract
    //deposit

    await registry.contract.deposit(registrAccount, { value: parseKlay('1') })
    const accBalance = await registry.contract.getBalance(registrAccount)
    expect(accBalance).to.be.equal(parseKlay('1'))

    // Request random words
    const l2RequestId = 1
    const txRequestRandomWords = await (
      await l1Endpoint.contract.requestRandomWords(
        keyHash,
        callbackGasLimit,
        SINGLE_WORD,
        registrAccount,
        l1Endpoint.signer.address, // consumer
        l2RequestId,
      )
    ).wait()
    expect(txRequestRandomWords.events.length).to.be.equal(5)
    const requestEvent = l1Endpoint.contract.interface.parseLog(txRequestRandomWords.events[4])
    expect(requestEvent.name).to.be.equal('RandomWordRequested')
    const numWords = SINGLE_WORD
    const sender = l1Endpoint.contract.address
    const isDirectPayment = true
    const { preSeed, accId, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinator.contract,
      keyHash,
      0,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment,
    )
    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      numWords,
    )

    const txFulfillRandomWords = await fulfillRandomWords(
      coordinator.contract,
      oracle,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment,
    )

    const fulfillEvent = l1Endpoint.contract.interface.parseLog(txFulfillRandomWords.events[0])
    expect(fulfillEvent.name).to.be.equal('RandomWordFulfilled')
    expect(fulfillEvent.args.sender).to.be.equal(l1Endpoint.signer.address)
  })

  it('Request & Fulfill Uint128', async function () {
    const requestFnName = 'requestDataUint128'
    const fulfillFnName = 'fulfillDataRequestUint128'
    const fulfillValue = [1]
    const fulfillEventName = 'DataRequestFulfilledUint128'
    const dataType = 'uint128'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })

  it('Request & Fulfill Int256', async function () {
    const requestFnName = 'requestDataInt256'
    const fulfillFnName = 'fulfillDataRequestInt256'
    const fulfillValue = [1]
    const fulfillEventName = 'DataRequestFulfilledInt256'
    const dataType = 'int256'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })

  it('Request & Fulfill Bool', async function () {
    const requestFnName = 'requestDataBool'
    const fulfillFnName = 'fulfillDataRequestBool'
    const fulfillValue = [true]
    const fulfillEventName = 'DataRequestFulfilledBool'
    const dataType = 'bool'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })

  it('Request & Fulfill String', async function () {
    const requestFnName = 'requestDataString'
    const fulfillFnName = 'fulfillDataRequestString'
    const fulfillValue = ['hello']
    const fulfillEventName = 'DataRequestFulfilledString'
    const dataType = 'string'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })

  it('Request & Fulfill Bytes32', async function () {
    const requestFnName = 'requestDataBytes32'
    const fulfillFnName = 'fulfillDataRequestBytes32'
    const fulfillValue = [ethers.utils.formatBytes32String('hello')]
    const fulfillEventName = 'DataRequestFulfilledBytes32'
    const dataType = 'bytes32'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })

  it('Request & Fulfill Bytes', async function () {
    const requestFnName = 'requestDataBytes'
    const fulfillFnName = 'fulfillDataRequestBytes'
    const fulfillValue = ['0x1234']
    const fulfillEventName = 'DataRequestFulfilledBytes'
    const dataType = 'bytes'
    const numSubmission = 1
    await requestAndFulfill(
      requestFnName,
      fulfillFnName,
      fulfillValue,
      fulfillEventName,
      dataType,
      numSubmission,
    )
  })
})
