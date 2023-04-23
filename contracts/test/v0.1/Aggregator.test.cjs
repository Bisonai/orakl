const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { aggregatorConfig } = require('./Aggregator.config.cjs')

async function contractBalance(contract) {
  return await ethers.provider.getBalance(contract)
}

async function createSigners() {
  let { deployer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
    await hre.getNamedAccounts()

  deployer = await ethers.getSigner(deployer)
  aggregatorOracle0 = await ethers.getSigner(aggregatorOracle0)
  aggregatorOracle1 = await ethers.getSigner(aggregatorOracle1)
  aggregatorOracle2 = await ethers.getSigner(aggregatorOracle2)

  return {
    deployer,
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

async function deploy() {
  const { deployer, aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } =
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

  // Deposit KLAY to Aggregator /////////////////////////////////////////////////
  const beforeBalance = await contractBalance(aggregator.address)
  expect(Number(beforeBalance)).to.be.equal(0)
  const value = ethers.utils.parseEther('1.0')
  await aggregator.deposit({ value })
  const afterBalance = await await contractBalance(aggregator.address)
  expect(afterBalance).to.be.equal(value)

  // Change oracles /////////////////////////////////////////////////////////////
  await changeOracles(aggregator, [aggregatorOracle0, aggregatorOracle1, aggregatorOracle2])

  return { aggregator, aggregatorProxy }
}

describe('Aggregator', function () {
  it('Should accept submissions', async function () {
    const { aggregator, aggregatorProxy } = await loadFixture(deploy)
    const { aggregatorOracle0, aggregatorOracle1, aggregatorOracle2 } = await createSigners()
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
    expect(Number(current1)).to.be.equal(10)
    expect(txReceipt1.events[2].event).to.be.equal('AvailableFundsUpdated')

    // third submission
    const txReceipt2 = await (await aggregator.connect(aggregatorOracle2).submit(1, 12)).wait()
    expect(txReceipt2.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt2.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current2 } = txReceipt2.events[1].args
    expect(Number(current2)).to.be.equal(11)
    expect(txReceipt2.events[2].event).to.be.equal('AvailableFundsUpdated')

    const withdrawablePayment0 = await aggregator.withdrawablePayment(aggregatorOracle0.address)
    const withdrawablePayment1 = await aggregator.withdrawablePayment(aggregatorOracle1.address)
    const withdrawablePayment2 = await aggregator.withdrawablePayment(aggregatorOracle2.address)

    expect(Number(withdrawablePayment0)).to.be.equal(paymentAmount)
    expect(Number(withdrawablePayment1)).to.be.equal(paymentAmount)
    expect(Number(withdrawablePayment2)).to.be.equal(paymentAmount)

    const { answer } = await aggregatorProxy.latestRoundData()
    expect(Number(answer)).to.be.equal(11)

    const proposedAggregator = await aggregatorProxy.proposedAggregator()
    expect(proposedAggregator).to.be.equal(ethers.constants.AddressZero)

    expect(await aggregatorProxy.aggregator()).to.be.equal(aggregator.address)
  })
})
