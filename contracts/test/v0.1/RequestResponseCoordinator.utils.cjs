const { expect } = require('chai')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')

const DATA_REQUEST_EVENT_ARGS = [
  'requestId',
  'jobId',
  'accId',
  'callbackGasLimit',
  'sender',
  'isDirectPayment',
  'data'
]

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
    blockHash
  }
}

module.exports = {
  setupOracle,
  parseDataRequestedTx,
  DATA_REQUEST_EVENT_ARGS
}
