const { expect } = require('chai')
const { ethers } = require('hardhat')
const { time, loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { deployAggregator } = require('./Aggregator.utils.cjs')
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
      restartDelay
    )
  ).wait()
}

async function deploy() {
  const { account0: deployerSigner, account1, account2, account3 } = await createSigners()

  // Aggregator /////////////////////////////////////////////////////////////////
  const aggregatorContract1 = await deployAggregator(deployerSigner)
  const aggregator1 = {
    contract: aggregatorContract1,
    signer: deployerSigner
  }

  const aggregatorContract2 = await deployAggregator(deployerSigner)
  const aggregator2 = {
    contract: aggregatorContract2,
    signer: deployerSigner
  }

  let batchSubmissionContract = await ethers.getContractFactory('BatchSubmission', {
    signer: deployerSigner
  })
  batchSubmissionContract = await batchSubmissionContract.deploy()
  await batchSubmissionContract.deployed()

  const batchSubmission = {
    contract: batchSubmissionContract,
    signer: deployerSigner
  }

  return {
    aggregator1,
    aggregator2,
    batchSubmission,
    account1,
    account2,
    account3
  }
}

describe('Batch submision', function () {
  it('Add & Remove Oracle', async function () {
    const {
      batchSubmission,
      account1: oracle0,
      account2: oracle1,
      account3: oracle2
    } = await loadFixture(deploy)

    // Add oracle ////////////////////////////////////////////////////////////
    await batchSubmission.contract.addOracle(oracle0.address)
    // Remove Oracle //////////////////////////////////////////////////////////
    // Cannot remove oracle that has not been added
    await expect(
      batchSubmission.contract.removeOracle(oracle1.address)
    ).to.be.revertedWithCustomError(batchSubmission.contract, 'InvalidOracle')
    const isOracleBeforeRemove = await batchSubmission.contract.oracleAddresses(oracle0.address)
    expect(isOracleBeforeRemove).to.be.equal(true)
    // Remove oracle that has been added before
    await batchSubmission.contract.removeOracle(oracle0.address)

    const isOracleAfterRemove = await batchSubmission.contract.oracleAddresses(oracle0.address)
    expect(isOracleAfterRemove).to.be.equal(false)
  })

  it('batch submit & read data', async function () {
    const {
      aggregator1,
      aggregator2,
      batchSubmission,
      account1: oracle0
    } = await loadFixture(deploy)
    const aggregators = [aggregator1.contract.address, aggregator2.contract.address]
    const rounds = [1, 1]
    const values = [10, 20]
    await expect(
      batchSubmission.contract.connect(oracle0).batchSubmit(aggregators, rounds, values)
    ).revertedWithCustomError(batchSubmission.contract, 'OnlyOracle')

    await changeOracles(aggregator1.contract, [], [batchSubmission.contract])
    await changeOracles(aggregator2.contract, [], [batchSubmission.contract])

    await batchSubmission.contract.addOracle(oracle0.address)
    const txReceipt0 = await (
      await batchSubmission.contract.connect(oracle0).batchSubmit(aggregators, rounds, values)
    ).wait()
    const iface = aggregator1.contract.interface
    const batchSubmisionIface = batchSubmission.contract.interface

    const events = txReceipt0.events.map((m, i) => {
      if (!m.event) return { ...iface.parseLog(txReceipt0.events[i]) }
      else return { ...batchSubmisionIface.parseLog(txReceipt0.events[i]) }
    })

    expect(events.length).to.be.equal(7)
    expect(events[6].name).to.be.equal('Submited')
    expect(events[6].args[0]).to.be.equal(2)

    expect(events[0].name).to.be.equal('NewRound')
    expect(events[1].name).to.be.equal('SubmissionReceived')

    expect(events[3].name).to.be.equal('NewRound')
    expect(events[4].name).to.be.equal('SubmissionReceived')

    const { submission: aggregatorCurrent1 } = events[1].args
    expect(aggregatorCurrent1).to.be.equal(10)

    const { submission: aggregatorCurrent2 } = events[4].args
    expect(aggregatorCurrent2).to.be.equal(20)
  })
})
