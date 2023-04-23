const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { aggregatorConfig } = require('./Aggregator.config.cjs')

async function contractBalance(contract) {
  return await ethers.provider.getBalance(contract)
}

async function createSigners() {
  let { deployer, consumer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
    await hre.getNamedAccounts()

  deployer = await ethers.getSigner(deployer)
  consumer = await ethers.getSigner(consumer)
  aggregatorOracle0 = await ethers.getSigner(aggregatorOracle0)
  aggregatorOracle1 = await ethers.getSigner(aggregatorOracle1)
  aggregatorOracle2 = await ethers.getSigner(aggregatorOracle2)

  return {
    deployer,
    consumer,
    aggregatorOracle0,
    aggregatorOracle1,
    aggregatorOracle2
  }
}

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
      restartDelay
    )
  ).wait()
}

async function deploy() {
  const { deployer, consumer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
    await createSigners()
  const { timeout, validator, decimals, description } = aggregatorConfig()

  // Aggregator /////////////////////////////////////////////////////////////////
  let aggregator = await ethers.getContractFactory('Aggregator', { signer: deployer.address })
  aggregator = await aggregator.deploy(timeout, validator, decimals, description)
  await aggregator.deployed()

  // AggregatorProxy ////////////////////////////////////////////////////////////
  let aggregatorProxy = await ethers.getContractFactory('AggregatorProxy', {
    signer: deployer.address
  })
  aggregatorProxy = await aggregatorProxy.deploy(aggregator.address)
  await aggregatorProxy.deployed()

  // Read configuration of Aggregator & AggregatorProxy
  expect(await aggregatorProxy.version()).to.be.equal(1)
  expect(await aggregatorProxy.description()).to.be.equal(description)

  // DataFeedConsumerMock ///////////////////////////////////////////////////////
  let dataFeedConsumerMock = await ethers.getContractFactory('DataFeedConsumerMock', {
    signer: consumer.address
  })
  dataFeedConsumerMock = await dataFeedConsumerMock.deploy(aggregatorProxy.address)
  await dataFeedConsumerMock.deployed()

  return { aggregator, aggregatorProxy, dataFeedConsumerMock }
}

describe('Aggregator', function () {
  it('Submit response', async function () {
    const { aggregator, aggregatorProxy, dataFeedConsumerMock } = await loadFixture(deploy)
    const { consumer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
      await createSigners()

    // Change oracles /////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [], [aggregatorOracle0, aggregatorOracle1, aggregatorOracle2])

    // First submission
    const txReceipt0 = await (await aggregator.connect(aggregatorOracle0).submit(1, 10)).wait()
    expect(txReceipt0.events.length).to.be.equal(2)
    expect(txReceipt0.events[0].event).to.be.equal('NewRound')
    expect(txReceipt0.events[1].event).to.be.equal('SubmissionReceived')

    // second submission
    const txReceipt1 = await (await aggregator.connect(aggregatorOracle1).submit(1, 11)).wait()
    expect(txReceipt1.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt1.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current1 } = txReceipt1.events[1].args
    expect(current1).to.be.equal(10)

    // third submission
    const txReceipt2 = await (await aggregator.connect(aggregatorOracle2).submit(1, 12)).wait()
    expect(txReceipt2.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt2.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current2 } = txReceipt2.events[1].args
    expect(current2).to.be.equal(11)

    const { answer } = await aggregatorProxy.latestRoundData()
    expect(answer).to.be.equal(11)

    const proposedAggregator = await aggregatorProxy.proposedAggregator()
    expect(proposedAggregator).to.be.equal(ethers.constants.AddressZero)

    expect(await aggregatorProxy.aggregator()).to.be.equal(aggregator.address)

    // Read submission from DataFeedConsumerMock ////////////////////////////////
    await dataFeedConsumerMock.getLatestRoundData()
    const sId = await dataFeedConsumerMock.sId()
    const sAnswer = await dataFeedConsumerMock.sAnswer()
    expect(sId).to.be.equal('18446744073709551617')
    expect(sAnswer).to.be.equal(11)

    // Read from aggregator proxy by specifying `roundID`
    const {
      id: pId,
      answer: pAnswer,
      startedAt: pStartedAt,
      updatedAt: pUpdatedAt,
      answeredInRound: pAnsweredInRound
    } = await aggregatorProxy.connect(consumer).getRoundData(sId)
    expect(pId).to.be.equal(sId)
    expect(pAnswer).to.be.equal(sAnswer)
    expect(pStartedAt).to.be.equal(await dataFeedConsumerMock.sStartedAt())
    expect(pUpdatedAt).to.be.equal(await dataFeedConsumerMock.sUpdatedAt())
    expect(pAnsweredInRound).to.be.equal(await dataFeedConsumerMock.sAnsweredInRound())

    // Read decimals from DataFeedConsumerMock //////////////////////////////////
    const { decimals } = aggregatorConfig()
    expect(await dataFeedConsumerMock.decimals()).to.be.equal(decimals)
  })

  it('Remove Oracle', async function () {
    const { aggregator } = await loadFixture(deploy)
    const { aggregatorOracle0, aggregatorOracle1 } = await createSigners()

    // Add 2 Oracles ////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [], [aggregatorOracle0, aggregatorOracle1])

    // Remove 1 Oracle //////////////////////////////////////////////////////////
    await changeOracles(aggregator, [aggregatorOracle0], [])

    const currentOracles = await aggregator.getOracles()
    expect(currentOracles.length).to.be.equal(1)
    expect(currentOracles[0]).to.be.equal(aggregatorOracle1.address)
  })

  it('addOracle assertions', async function () {
    const { aggregator } = await loadFixture(deploy)
    const { aggregatorOracle0, aggregatorOracle1 } = await createSigners()

    // Add Oracle ///////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [], [aggregatorOracle0])

    // Cannot add the same oracle twice
    await expect(
      aggregator.changeOracles([], [aggregatorOracle0.address], 1, 2, 0)
    ).to.be.revertedWithCustomError(aggregator, 'OracleAlreadyEnabled')
  })

  it('Propose & Confirm Aggregator Through AggregatorProxy', async function () {
    const {
      aggregator: currentAggregator,
      aggregatorProxy,
      dataFeedConsumerMock
    } = await loadFixture(deploy)
    const {
      deployer,
      consumer,
      aggregatorOracle0,
      aggregatorOracle1,
      aggregatorOracle2: invalidAggregator
    } = await createSigners()

    // Aggregator /////////////////////////////////////////////////////////////////
    const { timeout, validator, decimals, description } = aggregatorConfig()
    let aggregator = await ethers.getContractFactory('Aggregator', { signer: deployer.address })
    aggregator = await aggregator.deploy(timeout, validator, decimals, description)
    await aggregator.deployed()

    // Change oracles /////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [], [aggregatorOracle0, aggregatorOracle1])

    // proposeAggregator ////////////////////////////////////////////////////////
    // Aggregator can be proposed only by owner
    await expect(
      aggregatorProxy.connect(consumer).proposeAggregator(aggregator.address)
    ).to.be.revertedWith('Ownable: caller is not the owner')

    // Propose aggregator with contract owner
    const proposeAggregatorTx = await (
      await aggregatorProxy.proposeAggregator(aggregator.address)
    ).wait()
    expect(proposeAggregatorTx.events.length).to.be.equal(1)
    const proposeAggregatorEvent = aggregatorProxy.interface.parseLog(proposeAggregatorTx.events[0])
    expect(proposeAggregatorEvent.name).to.be.equal('AggregatorProposed')
    const { current, proposed } = proposeAggregatorEvent.args
    expect(current).to.be.equal(currentAggregator.address)
    expect(proposed).to.be.equal(aggregator.address)

    // proposedLatestRoundData //////////////////////////////////////////////////
    // If no data has been submitted to proposed yet, reading from proxy reverts
    await expect(aggregatorProxy.connect(consumer).proposedLatestRoundData()).to.be.revertedWith(
      'No data present'
    )
    await aggregator.connect(aggregatorOracle0).submit(1, 10)
    await aggregator.connect(aggregatorOracle1).submit(1, 10)

    // Read after submitting at least `minSubmissionCount` to proposed aggregator
    const { id, answer, startedAt, updatedAt, answeredInRound } = await aggregatorProxy
      .connect(consumer)
      .proposedLatestRoundData()
    expect(id).to.be.equal(1)
    expect(answer).to.be.equal(10)

    // Read from proposed aggregator by specifing `roundID`
    const {
      id: pId,
      answer: pAnswer,
      startedAt: pStartedAt,
      updatedAt: pUpdatedAt,
      answeredInRound: pAnsweredInRound
    } = await aggregatorProxy.connect(consumer).proposedGetRoundData(id)
    expect(id).to.be.equal(pId)
    expect(answer).to.be.equal(pAnswer)
    expect(startedAt).to.be.equal(pStartedAt)
    expect(updatedAt).to.be.equal(pUpdatedAt)
    expect(answeredInRound).to.be.equal(pAnsweredInRound)

    // confirmAggregator ////////////////////////////////////////////////////////
    // Aggregator can be confirmed only by owner
    expect(
      aggregatorProxy.connect(consumer).confirmAggregator(aggregator.address)
    ).to.be.revertedWith('Ownable: caller is not the owner')

    // Owner must pass proposed aggregator address, otherwise reverts
    await expect(
      aggregatorProxy.confirmAggregator(invalidAggregator.address)
    ).to.be.revertedWithCustomError(aggregatorProxy, 'InvalidProposedAggregator')

    // The initial `phaseId` is equal to 1
    const currentPhaseId = 1
    expect(await aggregatorProxy.phaseId()).to.be.equal(currentPhaseId)

    // Confirm aggregator with contract owner
    const confirmAggregatorTx = await (
      await aggregatorProxy.confirmAggregator(aggregator.address)
    ).wait()
    expect(confirmAggregatorTx.events.length).to.be.equal(1)
    const confirmAggregatorEvent = aggregatorProxy.interface.parseLog(confirmAggregatorTx.events[0])
    expect(confirmAggregatorEvent.name).to.be.equal('AggregatorConfirmed')
    const { previous, latest } = confirmAggregatorEvent.args
    expect(previous).to.be.equal(currentAggregator.address)
    expect(latest).to.be.equal(aggregator.address)

    // `phaseId` is increased by 1 after confirming the new aggregator
    expect(await aggregatorProxy.phaseId()).to.be.equal(currentPhaseId + 1)

    // Every Aggregator address that has been connected with
    // AggregatorProxy is stored in mapping and can be accessed through
    // `phaseAggregators` by specifing the `phaseId`.
    expect(await aggregatorProxy.phaseAggregators(1)).to.be.equal(current)
    expect(await aggregatorProxy.phaseAggregators(2)).to.be.equal(proposed)
  })
})
