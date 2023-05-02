const { expect } = require('chai')
const { aggregatorConfig } = require('./Aggregator.config.cjs')

async function deployAggregator(signer) {
  const { timeout, validator, decimals, description } = aggregatorConfig()
  let aggregator = await ethers.getContractFactory('Aggregator', { signer })
  aggregator = await aggregator.deploy(timeout, validator, decimals, description)
  await aggregator.deployed()
  return aggregator
}

function parseSetRequesterPermissionsTx(aggregator, tx) {
  expect(tx.events.length).to.be.equal(1)
  expect(tx.events[0].event).to.be.equal('RequesterPermissionsSet')
  const removeRequesterPermissionsEvent = aggregator.interface.parseLog(tx.events[0])
  const { requester, authorized, delay } = removeRequesterPermissionsEvent.args

  return { requester, authorized, delay }
}

module.exports = {
  deployAggregator,
  parseSetRequesterPermissionsTx
}
