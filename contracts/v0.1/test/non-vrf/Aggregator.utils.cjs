const { expect } = require('chai')
const { aggregatorConfig } = require('./Aggregator.config.cjs')

async function deployAggregator(signer) {
  const { timeout, validator, decimals, description } = aggregatorConfig()
  let contract = await ethers.getContractFactory('Aggregator', { signer })
  contract = await contract.deploy(timeout, validator, decimals, description)
  await contract.deployed()
  return contract
}

async function deployAggregatorProxy(aggregatorAddress, signer) {
  let contract = await ethers.getContractFactory('AggregatorProxy', {
    signer,
  })
  contract = await contract.deploy(aggregatorAddress)
  await contract.deployed()
  return contract
}

async function deployDataFeedConsumerMock(aggregatorProxyAddress, signer) {
  let contract = await ethers.getContractFactory('DataFeedConsumerMock', {
    signer,
  })
  contract = await contract.deploy(aggregatorProxyAddress)
  await contract.deployed()
  return contract
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
  parseSetRequesterPermissionsTx,
  deployAggregatorProxy,
  deployDataFeedConsumerMock,
}
