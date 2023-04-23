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

async function changeOracles(aggregator, oracles) {
  const removed = []
  const added = oracles.map((x) => x.address)
  const addedAdmins = added
  const minSubmissionCount = 2
  const maxSubmissionCount = oracles.length
  const restartDelay = 0

  await aggregator.changeOracles(
    removed,
    added,
    addedAdmins,
    minSubmissionCount,
    maxSubmissionCount,
    restartDelay
  )
}

async function depositToAggregator(aggregator) {
  const beforeBalance = await contractBalance(aggregator.address)
  expect(Number(beforeBalance)).to.be.equal(0)
  const value = ethers.utils.parseEther('1.0')
  await aggregator.deposit({ value })
  const afterBalance = await await contractBalance(aggregator.address)
  expect(afterBalance).to.be.equal(value)
}

async function deploy() {
  const { deployer, consumer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
    await createSigners()
  const { paymentAmount, timeout, validator, decimals, description } = aggregatorConfig()

  // Aggregator /////////////////////////////////////////////////////////////////
  let aggregator = await ethers.getContractFactory('Aggregator', { signer: deployer.address })
  aggregator = await aggregator.deploy(paymentAmount, timeout, validator, decimals, description)
  await aggregator.deployed()

  // AggregatorProxy ////////////////////////////////////////////////////////////
  let aggregatorProxy = await ethers.getContractFactory('AggregatorProxy', {
    signer: deployer.address
  })
  aggregatorProxy = await aggregatorProxy.deploy(aggregator.address)
  await aggregatorProxy.deployed()

  // DataFeedConsumerMock ///////////////////////////////////////////////////////
  let dataFeedConsumerMock = await ethers.getContractFactory('DataFeedConsumerMock', {
    signer: consumer.address
  })
  dataFeedConsumerMock = await dataFeedConsumerMock.deploy(aggregatorProxy.address)
  await dataFeedConsumerMock.deployed()

  // Deposit KLAY to Aggregator /////////////////////////////////////////////////
  await depositToAggregator(aggregator)

  // Change oracles /////////////////////////////////////////////////////////////
  await changeOracles(aggregator, [aggregatorOracle0, aggregatorOracle1, aggregatorOracle2])

  return { aggregator, aggregatorProxy, dataFeedConsumerMock }
}

describe('Aggregator', function () {
  it('Submit response', async function () {
    const { aggregator, aggregatorProxy, dataFeedConsumerMock } = await loadFixture(deploy)
    const { consumer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
      await createSigners()
    const { paymentAmount } = aggregatorConfig()

    // First submission
    const txReceipt0 = await (await aggregator.connect(aggregatorOracle0).submit(1, 10)).wait()
    expect(txReceipt0.events.length).to.be.equal(3)
    expect(txReceipt0.events[0].event).to.be.equal('NewRound')
    expect(txReceipt0.events[1].event).to.be.equal('SubmissionReceived')
    expect(txReceipt0.events[2].event).to.be.equal('AvailableFundsUpdated')

    // second submission
    const txReceipt1 = await (await aggregator.connect(aggregatorOracle1).submit(1, 11)).wait()
    expect(txReceipt1.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt1.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current1 } = txReceipt1.events[1].args
    expect(current1).to.be.equal(10)
    expect(txReceipt1.events[2].event).to.be.equal('AvailableFundsUpdated')

    // third submission
    const txReceipt2 = await (await aggregator.connect(aggregatorOracle2).submit(1, 12)).wait()
    expect(txReceipt2.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt2.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current2 } = txReceipt2.events[1].args
    expect(current2).to.be.equal(11)
    expect(txReceipt2.events[2].event).to.be.equal('AvailableFundsUpdated')

    const withdrawablePayment0 = await aggregator.withdrawablePayment(aggregatorOracle0.address)
    const withdrawablePayment1 = await aggregator.withdrawablePayment(aggregatorOracle1.address)
    const withdrawablePayment2 = await aggregator.withdrawablePayment(aggregatorOracle2.address)

    expect(withdrawablePayment0).to.be.equal(paymentAmount)
    expect(withdrawablePayment1).to.be.equal(paymentAmount)
    expect(withdrawablePayment2).to.be.equal(paymentAmount)

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
    const { paymentAmount, timeout, validator, decimals, description } = aggregatorConfig()
    let aggregator = await ethers.getContractFactory('Aggregator', { signer: deployer.address })
    aggregator = await aggregator.deploy(paymentAmount, timeout, validator, decimals, description)
    await aggregator.deployed()

    // Deposit $KLAY to Aggregator contract /////////////////////////////////////
    await depositToAggregator(aggregator)

    // Change oracles /////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [aggregatorOracle0, aggregatorOracle1])

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
  })
})
