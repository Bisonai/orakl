const { expect } = require('chai')
const { ethers } = require('hardhat')
const { time, loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { aggregatorConfig } = require('./Aggregator.config.cjs')
const {
  deployAggregatorProxy,
  deployAggregator,
  deployDataFeedConsumerMock,
} = require('./Aggregator.utils.cjs')
const { createSigners } = require('../utils.cjs')

async function changeOracles(aggregator, removeOracles, addOracles) {
  const currentOracles = await aggregator.getOracles()

  const removed = removeOracles.map((x) => x.address)
  const added = addOracles.map((x) => x.address)
  const maxSubmissionCount = currentOracles.length + addOracles.length - removeOracles.length
  const minSubmissionCount = Math.min(2, maxSubmissionCount)
  const restartDelay = 0

  return await (
    await aggregator.changeOracles(
      removed,
      added,
      minSubmissionCount,
      maxSubmissionCount,
      restartDelay,
    )
  ).wait()
}

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2,
    account3,
    account4,
    account5,
  } = await createSigners()

  // Aggregator /////////////////////////////////////////////////////////////////
  const aggregatorContract = await deployAggregator(deployerSigner)
  const aggregator = {
    contract: aggregatorContract,
    signer: deployerSigner,
  }

  // AggregatorProxy ////////////////////////////////////////////////////////////
  const aggregatorProxyContract = await deployAggregatorProxy(
    aggregator.contract.address,
    deployerSigner,
  )
  const aggregatorProxy = {
    contract: aggregatorProxyContract,
    signer: deployerSigner,
  }

  // Read configuration of Aggregator & AggregatorProxy
  const { description } = aggregatorConfig()
  expect(await aggregatorProxy.contract.typeAndVersion()).to.be.equal('Aggregator v0.1')
  expect(await aggregatorProxy.contract.description()).to.be.equal(description)

  // DataFeedConsumerMock ///////////////////////////////////////////////////////
  const consumerContract = await deployDataFeedConsumerMock(
    aggregatorProxy.contract.address,
    consumerSigner,
  )
  const consumer = {
    contract: consumerContract,
    signer: consumerSigner,
  }

  // L2 endpoint
  let l2EndpointContract = await ethers.getContractFactory('L2Endpoint', { deployerSigner })
  l2EndpointContract = await l2EndpointContract.deploy()
  await l2EndpointContract.deployed()

  const endpoint = {
    contract: l2EndpointContract,
    signer: deployerSigner,
  }

  return {
    aggregator,
    aggregatorProxy,
    endpoint,
    consumer,
    account2,
    account3,
    account4,
    account5,
  }
}

describe('L2Endpoint', function () {
  it('Add and remove aggregator,submitter', async function () {
    const { aggregator, endpoint } = await loadFixture(deploy)
    await endpoint.contract.addAggregator(aggregator.contract.address)
    await endpoint.contract.addSubmitter(endpoint.signer.address)

    let aggreatorCount = await endpoint.contract.sAggregatorCount()
    expect(aggreatorCount).to.be.equal(1)

    let submitterCount = await endpoint.contract.sSubmitterCount()
    expect(submitterCount).to.be.equal(1)

    await endpoint.contract.removeAggregator(aggregator.contract.address)
    await endpoint.contract.removeSubmitter(endpoint.signer.address)

    aggreatorCount = await endpoint.contract.sAggregatorCount()
    expect(aggreatorCount).to.be.equal(0)

    submitterCount = await endpoint.contract.sSubmitterCount()
    expect(submitterCount).to.be.equal(0)
  })

  it('Add and remove invalid aggregator,submitter ', async function () {
    const { aggregator, endpoint } = await loadFixture(deploy)
    await endpoint.contract.addAggregator(aggregator.contract.address)
    await expect(
      endpoint.contract.addAggregator(aggregator.contract.address),
    ).to.be.revertedWithCustomError(endpoint.contract, 'InvalidAggregator')

    await endpoint.contract.addSubmitter(endpoint.signer.address)
    await expect(
      endpoint.contract.addSubmitter(endpoint.signer.address),
    ).to.be.revertedWithCustomError(endpoint.contract, 'InvalidSubmitter')

    let aggreatorCount = await endpoint.contract.sAggregatorCount()
    expect(aggreatorCount).to.be.equal(1)

    let submitterCount = await endpoint.contract.sSubmitterCount()
    expect(submitterCount).to.be.equal(1)

    await endpoint.contract.removeAggregator(aggregator.contract.address)
    await expect(
      endpoint.contract.removeAggregator(aggregator.contract.address),
    ).to.be.revertedWithCustomError(endpoint.contract, 'InvalidAggregator')

    await endpoint.contract.removeSubmitter(endpoint.signer.address)
    await expect(
      endpoint.contract.removeSubmitter(endpoint.signer.address),
    ).to.be.revertedWithCustomError(endpoint.contract, 'InvalidSubmitter')

    aggreatorCount = await endpoint.contract.sAggregatorCount()
    expect(aggreatorCount).to.be.equal(0)

    submitterCount = await endpoint.contract.sSubmitterCount()
    expect(submitterCount).to.be.equal(0)
  })

  it('Submit & Read Response', async function () {
    const { aggregator, aggregatorProxy, endpoint, consumer } = await loadFixture(deploy)
    await changeOracles(aggregator.contract, [], [endpoint.contract])
    await endpoint.contract.addAggregator(aggregator.contract.address)
    await endpoint.contract.addSubmitter(endpoint.signer.address)
    //First submission
    const txReceipt0 = await (
      await endpoint.contract.connect(endpoint.signer).submit(1, 10, aggregator.contract.address)
    ).wait()
    expect(txReceipt0.events.length).to.be.equal(4)
    expect(txReceipt0.events[3].event).to.be.equal('Submitted')

    const { answer } = await aggregatorProxy.contract.latestRoundData()
    expect(answer).to.be.equal(10)

    // Read submission from DataFeedConsumerMock ////////////////////////////////
    await consumer.contract.getLatestRoundData()
    const sAnswer = await consumer.contract.sAnswer()
    expect(sAnswer).to.be.equal(10)
  })
})
