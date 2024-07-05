const { expect } = require('chai')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')

const DATA_REQUEST_EVENT_ARGS = [
  'requestId',
  'jobId',
  'accId',
  'callbackGasLimit',
  'sender',
  'isDirectPayment',
  'data',
]

async function deploy(prepaymentAddress, signer) {
  let contract = await ethers.getContractFactory('RequestResponseCoordinator', {
    signer,
  })
  contract = await contract.deploy(prepaymentAddress)
  await contract.deployed()
  return contract
}

async function setupOracle(coordinator, oracle) {
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()
  await coordinator.registerOracle(oracle)
  await coordinator.setConfig(maxGasLimit, gasAfterPaymentCalculation, Object.values(feeConfig))
}

function parseDataRequestedTx(coordinator, tx) {
  expect(tx.events.length).to.be.equal(1)
  const event = coordinator.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('DataRequested')
  const { requestId, jobId, accId, callbackGasLimit, sender, isDirectPayment, data } = event.args
  const blockNumber = tx.blockNumber
  const blockHash = tx.blockHash

  for (const arg of DATA_REQUEST_EVENT_ARGS) {
    expect(event.args[arg]).to.not.be.undefined
  }

  return {
    requestId,
    jobId,
    accId,
    callbackGasLimit,
    sender,
    isDirectPayment,
    data,
    blockNumber,
    blockHash,
  }
}

function parseDataRequestFulfilledTx(coordinator, tx, eventName) {
  const event = coordinator.interface.parseLog(tx.events[tx.events.length - 1])
  expect(event.name).to.be.equal(eventName)
  const blockHash = tx.blockHash
  const blockNumber = tx.blockNumber
  const gasUsed = tx.gasUsed
  const cumulativeGasUsed = tx.cumulativeGasUsed

  const { requestId, response, payment, success } = event.args
  return {
    requestId,
    response,
    payment,
    success,
    blockHash,
    blockNumber,
    gasUsed,
    cumulativeGasUsed,
  }
}

function parseOracleRegisterdTx(coordinator, tx) {
  expect(tx.events.length).to.be.equal(1)
  const event = coordinator.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('OracleRegistered')
  const { oracle } = event.args
  return { oracle }
}

module.exports = {
  deploy,
  setupOracle,
  parseDataRequestedTx,
  DATA_REQUEST_EVENT_ARGS,
  parseDataRequestFulfilledTx,
  parseOracleRegisterdTx,
}
