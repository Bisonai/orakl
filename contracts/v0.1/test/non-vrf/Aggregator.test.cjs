const { expect } = require('chai')
const { ethers } = require('hardhat')
const { time, loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { aggregatorConfig } = require('./Aggregator.config.cjs')
const {
  deployAggregatorProxy,
  deployAggregator,
  parseSetRequesterPermissionsTx,
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

  return {
    aggregator,
    aggregatorProxy,
    consumer,
    account2,
    account3,
    account4,
    account5,
  }
}

describe('Aggregator', function () {
  it('Add & Remove Oracle', async function () {
    const {
      aggregator,
      account2: oracle0,
      account3: oracle1,
      account4: oracle2,
    } = await loadFixture(deploy)

    // Add 2 Oracles ////////////////////////////////////////////////////////////
    await changeOracles(aggregator.contract, [], [oracle0, oracle1])

    // Remove Oracle //////////////////////////////////////////////////////////
    // Cannot remove oracle that has not been added
    await expect(
      aggregator.contract.changeOracles([oracle2.address], [], 1, 1, 0),
    ).to.be.revertedWithCustomError(aggregator.contract, 'OracleNotEnabled')

    // Remove oracle that has been added before
    await changeOracles(aggregator.contract, [oracle0], [])

    const currentOracles = await aggregator.contract.getOracles()
    expect(currentOracles.length).to.be.equal(1)
    expect(currentOracles[0]).to.be.equal(oracle1.address)
  })

  it('Submit & Read Response', async function () {
    const {
      aggregator,
      aggregatorProxy,
      consumer,
      account2: oracle0,
      account3: oracle1,
      account4: oracle2,
    } = await loadFixture(deploy)

    // Change oracles /////////////////////////////////////////////////////////////
    await changeOracles(aggregator.contract, [], [oracle0, oracle1, oracle2])

    // First submission
    const txReceipt0 = await (await aggregator.contract.connect(oracle0).submit(1, 10)).wait()
    expect(txReceipt0.events.length).to.be.equal(2)
    expect(txReceipt0.events[0].event).to.be.equal('NewRound')
    expect(txReceipt0.events[1].event).to.be.equal('SubmissionReceived')

    // second submission
    const txReceipt1 = await (await aggregator.contract.connect(oracle1).submit(1, 11)).wait()
    expect(txReceipt1.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt1.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current1 } = txReceipt1.events[1].args
    expect(current1).to.be.equal(10)

    // third submission
    const txReceipt2 = await (await aggregator.contract.connect(oracle2).submit(1, 12)).wait()
    expect(txReceipt2.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt2.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current2 } = txReceipt2.events[1].args
    expect(current2).to.be.equal(11)

    const { answer } = await aggregatorProxy.contract.latestRoundData()
    expect(answer).to.be.equal(11)

    const proposedAggregator = await aggregatorProxy.contract.proposedAggregator()
    expect(proposedAggregator).to.be.equal(ethers.constants.AddressZero)

    // Address of current aggregator can be obtained through
    // AggregatorProxy `aggregator` function.
    expect(await aggregatorProxy.contract.aggregator()).to.be.equal(aggregator.contract.address)

    // Read submission from DataFeedConsumerMock ////////////////////////////////
    await consumer.contract.getLatestRoundData()
    const sId = await consumer.contract.sId()
    const sAnswer = await consumer.contract.sAnswer()
    expect(sId).to.be.equal('18446744073709551617')
    expect(sAnswer).to.be.equal(11)

    // Read from aggregator proxy by specifying `roundID`
    const {
      id: pId,
      answer: pAnswer,
      startedAt: pStartedAt,
      updatedAt: pUpdatedAt,
      answeredInRound: pAnsweredInRound,
    } = await aggregatorProxy.contract.connect(consumer.signer).getRoundData(sId)
    expect(pId).to.be.equal(sId)
    expect(pAnswer).to.be.equal(sAnswer)
    expect(pStartedAt).to.be.equal(await consumer.contract.sStartedAt())
    expect(pUpdatedAt).to.be.equal(await consumer.contract.sUpdatedAt())
    expect(pAnsweredInRound).to.be.equal(await consumer.contract.sAnsweredInRound())

    // Read decimals from DataFeedConsumerMock //////////////////////////////////
    const { decimals } = aggregatorConfig()
    expect(await consumer.contract.decimals()).to.be.equal(decimals)
  })

  it('addOracle assertions', async function () {
    const { aggregator, account2: oracle0, account3: oracle1 } = await loadFixture(deploy)

    // Add Oracle ///////////////////////////////////////////////////////////////
    await changeOracles(aggregator.contract, [], [oracle0])

    // Cannot add the same oracle twice
    await expect(
      aggregator.contract.changeOracles([], [oracle0.address], 1, 2, 0),
    ).to.be.revertedWithCustomError(aggregator.contract, 'OracleAlreadyEnabled')
  })

  it('Propose & Confirm Aggregator Through AggregatorProxy', async function () {
    const {
      aggregator: currentAggregator,
      aggregatorProxy,
      consumer,
      account2: oracle0,
      account3: oracle1,
      account4: invalidOracle,
    } = await loadFixture(deploy)

    // Aggregator /////////////////////////////////////////////////////////////////
    const aggregator = await deployAggregator(currentAggregator.deployer)

    // Change oracles /////////////////////////////////////////////////////////////
    await changeOracles(aggregator, [], [oracle0, oracle1])

    // proposeAggregator ////////////////////////////////////////////////////////
    // Aggregator can be proposed only by owner
    await expect(
      aggregatorProxy.contract.connect(consumer.signer).proposeAggregator(aggregator.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    // Propose aggregator with contract owner
    const proposeAggregatorTx = await (
      await aggregatorProxy.contract.proposeAggregator(aggregator.address)
    ).wait()
    expect(proposeAggregatorTx.events.length).to.be.equal(1)
    const proposeAggregatorEvent = aggregatorProxy.contract.interface.parseLog(
      proposeAggregatorTx.events[0],
    )
    expect(proposeAggregatorEvent.name).to.be.equal('AggregatorProposed')
    const { current, proposed } = proposeAggregatorEvent.args
    expect(current).to.be.equal(currentAggregator.contract.address)
    expect(proposed).to.be.equal(aggregator.address)

    // proposedLatestRoundData //////////////////////////////////////////////////
    // If no data has been submitted to proposed yet, reading from proxy reverts
    await expect(
      aggregatorProxy.contract.connect(consumer.signer).proposedLatestRoundData(),
    ).to.be.revertedWithCustomError(aggregator, 'NoDataPresent')
    await aggregator.connect(oracle0).submit(1, 10)
    await aggregator.connect(oracle1).submit(1, 10)

    // Read after submitting at least `minSubmissionCount` to proposed aggregator
    const { id, answer, startedAt, updatedAt, answeredInRound } = await aggregatorProxy.contract
      .connect(consumer.signer)
      .proposedLatestRoundData()
    expect(id).to.be.equal(1)
    expect(answer).to.be.equal(10)

    // Read from proposed aggregator by specifing `roundID`
    const {
      id: pId,
      answer: pAnswer,
      startedAt: pStartedAt,
      updatedAt: pUpdatedAt,
      answeredInRound: pAnsweredInRound,
    } = await aggregatorProxy.contract.connect(consumer.signer).proposedGetRoundData(id)
    expect(id).to.be.equal(pId)
    expect(answer).to.be.equal(pAnswer)
    expect(startedAt).to.be.equal(pStartedAt)
    expect(updatedAt).to.be.equal(pUpdatedAt)
    expect(answeredInRound).to.be.equal(pAnsweredInRound)

    // confirmAggregator ////////////////////////////////////////////////////////
    // Aggregator can be confirmed only by owner
    expect(
      aggregatorProxy.contract.connect(consumer.signer).confirmAggregator(aggregator.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    // Owner must pass proposed aggregator address, otherwise reverts
    await expect(
      aggregatorProxy.contract.confirmAggregator(invalidOracle.address),
    ).to.be.revertedWithCustomError(aggregatorProxy.contract, 'InvalidProposedAggregator')

    // The initial `phaseId` is equal to 1
    const currentPhaseId = 1
    expect(await aggregatorProxy.contract.phaseId()).to.be.equal(currentPhaseId)

    // Confirm aggregator with contract owner
    {
      const tx = await (await aggregatorProxy.contract.confirmAggregator(aggregator.address)).wait()
      expect(tx.events.length).to.be.equal(1)
      const event = aggregatorProxy.contract.interface.parseLog(tx.events[0])
      expect(event.name).to.be.equal('AggregatorConfirmed')
      const { previous, latest } = event.args
      expect(previous).to.be.equal(currentAggregator.contract.address)
      expect(latest).to.be.equal(aggregator.address)
    }

    // `phaseId` is increased by 1 after confirming the new aggregator
    expect(await aggregatorProxy.contract.phaseId()).to.be.equal(currentPhaseId + 1)

    // Every Aggregator address that has been connected with
    // AggregatorProxy is stored in mapping and can be accessed through
    // `phaseAggregators` by specifing the `phaseId`.
    expect(await aggregatorProxy.contract.phaseAggregators(1)).to.be.equal(current)
    expect(await aggregatorProxy.contract.phaseAggregators(2)).to.be.equal(proposed)
  })

  it('oracleRoundState', async function () {
    const { aggregator, account2: oracle0, account3: oracle1 } = await loadFixture(deploy)

    // Add Oracle ///////////////////////////////////////////////////////////////
    await changeOracles(aggregator.contract, [], [oracle0])

    {
      // State of oracle before the first submission
      const { _roundId, _latestSubmission, _startedAt, _timeout, _oracleCount } =
        await aggregator.contract.oracleRoundState(oracle0.address, 0)
      expect(_roundId).to.be.equal(1)
      expect(_latestSubmission).to.be.equal(0)
      expect(_startedAt).to.be.equal(0)
      expect(_timeout).to.be.equal(0)
      expect(_oracleCount).to.be.equal(1)
    }

    // Submit to aggregator
    const roundId = 1
    const submission = 10
    await aggregator.contract.connect(oracle0).submit(roundId, submission)

    // State of oracle after the first submission
    {
      const { _roundId, _latestSubmission, _oracleCount } =
        await aggregator.contract.oracleRoundState(oracle0.address, roundId)
      expect(_roundId).to.be.equal(roundId)
      expect(_latestSubmission).to.be.equal(submission)
      expect(_oracleCount).to.be.equal(1)
    }
  })

  it('External Requester', async function () {
    const {
      aggregator,
      account2: authorizedRequester,
      account3: unauthorizedRequester,
    } = await loadFixture(deploy)

    const requesterAddress = authorizedRequester.address

    // Add a new requester //////////////////////////////////////////////////////
    {
      const _authorized = true
      const _delay = 0
      const tx = await (
        await aggregator.contract.setRequesterPermissions(requesterAddress, _authorized, _delay)
      ).wait()
      const { requester, authorized, delay } = parseSetRequesterPermissionsTx(
        aggregator.contract,
        tx,
      )
      expect(requester).to.be.equal(authorizedRequester.address)
      expect(authorized).to.be.equal(_authorized)
      expect(delay).to.be.equal(_delay)
    }

    {
      const _authorized = true
      const _delay = 0
      // Test idempotency for adding a new requester
      const tx = await (
        await aggregator.contract.setRequesterPermissions(requesterAddress, _authorized, _delay)
      ).wait()
      // No new requester added -> no emmited event
      expect(tx.events.length).to.be.equal(0)
    }

    // Request NewRound /////////////////////////////////////////////////////////
    // Only authorized requester can request new round, otherwise reverts
    await expect(
      aggregator.contract.connect(unauthorizedRequester).requestNewRound(),
    ).to.be.revertedWithCustomError(aggregator.contract, 'RequesterNotAuthorized')

    // Request with authorized requester
    {
      const tx = await (
        await aggregator.contract.connect(authorizedRequester).requestNewRound()
      ).wait()
      const blockTimestamp = (await ethers.provider.getBlock(tx.blockNumber)).timestamp
      expect(tx.events.length).to.be.equal(1)
      expect(tx.events[0].event).to.be.equal('NewRound')
      const event = aggregator.contract.interface.parseLog(tx.events[0])
      const { roundId, startedBy, startedAt } = event.args
      expect(roundId).to.be.equal(1)
      expect(startedBy).to.be.equal(authorizedRequester.address)
      expect(startedAt).to.be.equal(blockTimestamp)
    }

    // Remove requester /////////////////////////////////////////////////////////
    {
      const _authorized = false
      const _delay = 0
      const tx = await (
        await aggregator.contract.setRequesterPermissions(
          authorizedRequester.address,
          _authorized,
          _delay,
        )
      ).wait()
      const { requester, authorized, delay } = parseSetRequesterPermissionsTx(
        aggregator.contract,
        tx,
      )
      expect(requester).to.be.equal(authorizedRequester.address)
      expect(authorized).to.be.equal(_authorized)
      expect(delay).to.be.equal(_delay)
    }
  })

  it('TooManyOracles', async function () {
    const { aggregator, consumer } = await loadFixture(deploy)

    const MAX_ORACLE_COUNT = (
      await aggregator.contract.connect(consumer.signer).MAX_ORACLE_COUNT()
    ).toNumber()

    let i = 1
    for (; i < MAX_ORACLE_COUNT; ++i) {
      const { address: oracle } = ethers.Wallet.createRandom()
      await aggregator.contract.changeOracles([], [oracle], i, i, 0)
    }

    const { address: oracle } = ethers.Wallet.createRandom()
    await expect(
      aggregator.contract.changeOracles([], [oracle], i, i, 0),
    ).to.be.revertedWithCustomError(aggregator.contract, 'TooManyOracles')
  })

  it('MinSubmissionGtMaxSubmission', async function () {
    const { aggregator, consumer } = await loadFixture(deploy)
    const minSubmissionCount = 1
    const maxSubmissionCount = 0
    await expect(
      aggregator.contract.changeOracles([], [], minSubmissionCount, maxSubmissionCount, 0),
    ).to.be.revertedWithCustomError(aggregator.contract, 'MinSubmissionGtMaxSubmission')
  })

  it('MaxSubmissionGtOracleNum', async function () {
    const { aggregator, consumer } = await loadFixture(deploy)
    const minSubmissionCount = 0
    const maxSubmissionCount = 1
    await expect(
      aggregator.contract.changeOracles([], [], minSubmissionCount, maxSubmissionCount, 0),
    ).to.be.revertedWithCustomError(aggregator.contract, 'MaxSubmissionGtOracleNum')
  })

  it('RestartDelayExceedOracleNum', async function () {
    const { aggregator, consumer } = await loadFixture(deploy)
    const minSubmissionCount = 0
    const maxSubmissionCount = 1
    const restartDelay = 1
    const { address: oracle } = ethers.Wallet.createRandom()
    await expect(
      aggregator.contract.changeOracles(
        [],
        [oracle],
        minSubmissionCount,
        maxSubmissionCount,
        restartDelay,
      ),
    ).to.be.revertedWithCustomError(aggregator.contract, 'RestartDelayExceedOracleNum')
  })

  it('MinSubmissionZero', async function () {
    const { aggregator, consumer } = await loadFixture(deploy)
    const minSubmissionCount = 0
    const maxSubmissionCount = 1
    const restartDelay = 0
    const { address: oracle } = ethers.Wallet.createRandom()
    await expect(
      aggregator.contract.changeOracles(
        [],
        [oracle],
        minSubmissionCount,
        maxSubmissionCount,
        restartDelay,
      ),
    ).to.be.revertedWithCustomError(aggregator.contract, 'MinSubmissionZero')
  })

  it('PrevRoundNotSupersedable', async function () {
    const {
      aggregator,
      consumer,
      account2: requester,
      account3: oracle0,
      account4: oracle1,
      account5: oracle2,
    } = await loadFixture(deploy)
    const authorized = true
    const delay = 0
    await aggregator.contract.setRequesterPermissions(requester.address, authorized, delay)

    await aggregator.contract.changeOracles(
      [],
      [oracle0.address, oracle1.address, oracle2.address],
      2,
      3,
      0,
    )

    // First round
    await aggregator.contract.connect(oracle0).submit(1, 123)
    // Only single oracle submitted, but did not compute answer (requires at least two submissions)

    await expect(
      aggregator.contract.connect(requester).requestNewRound(),
    ).to.be.revertedWithCustomError(aggregator.contract, 'PrevRoundNotSupersedable')
  })

  it('currentRoundStartedAt', async function () {
    const { aggregator, consumer, account2: oracle0 } = await loadFixture(deploy)
    await aggregator.contract.changeOracles([], [oracle0.address], 1, 1, 0)

    for (let i = 1; i <= 2; ++i) {
      const tx = await (await aggregator.contract.connect(oracle0).submit(i, 123)).wait()
      const block = await ethers.provider.getBlock(tx.blockNumber)
      const startedAt = await aggregator.contract.connect(consumer.signer).currentRoundStartedAt()
      expect(startedAt).to.be.equal(block.timestamp)
    }
  })

  it('validateOracleRound', async function () {
    const { aggregator, consumer, account2: oracle0, account3: oracle1 } = await loadFixture(deploy)
    const answer = 123

    {
      const roundId = 1
      await expect(aggregator.contract.connect(oracle0).submit(roundId, answer)).to.be.revertedWith(
        'not enabled oracle',
      )
    }

    await aggregator.contract.changeOracles([], [oracle0.address], 1, 1, 0)

    {
      const roundId = 2
      await expect(aggregator.contract.connect(oracle0).submit(roundId, answer)).to.be.revertedWith(
        'invalid round to report',
      )
    }

    {
      const roundId = 1
      await aggregator.contract.connect(oracle0).submit(roundId, answer)
      await expect(aggregator.contract.connect(oracle0).submit(roundId, answer)).to.be.revertedWith(
        'cannot report on previous rounds',
      )
    }

    await aggregator.contract.changeOracles([], [oracle1.address], 2, 2, 0)

    {
      const roundId = 2
      await aggregator.contract.connect(oracle0).submit(roundId, answer)
      await expect(
        aggregator.contract.connect(oracle0).submit(roundId + 1, answer),
      ).to.be.revertedWith('previous round not supersedable')
    }

    {
      const { timeout } = aggregatorConfig()
      time.increase(timeout)
      await aggregator.contract.changeOracles([oracle1.address], [], 1, 1, 0)

      const roundId = 3
      await aggregator.contract.connect(oracle1).submit(roundId, answer)
      await expect(
        aggregator.contract.connect(oracle1).submit(roundId + 1, answer),
      ).to.be.revertedWith('no longer allowed oracle')
    }
  })

  it('Skipping rounds', async function () {
    const { aggregator, account2: oracle0, account3: oracle1 } = await loadFixture(deploy)

    const timeout = 0
    await aggregator.contract.changeOracles([], [oracle0.address, oracle1.address], 1, 2, timeout)

    {
      // oracle 0, 1
      // round 1
      const round = 1
      await aggregator.contract.connect(oracle0).submit(round, 123)
      await aggregator.contract.connect(oracle1).submit(round, 123)
    }

    {
      // oracle 0
      // round 2
      const { _eligibleToSubmit, _roundId } = await aggregator.contract.oracleRoundState(
        oracle0.address,
        0,
      )
      expect(_roundId).to.be.equal(2)
      await aggregator.contract.connect(oracle0).submit(_roundId, 123)
    }

    {
      // oracle 0
      // round 3
      const { _eligibleToSubmit, _roundId } = await aggregator.contract.oracleRoundState(
        oracle0.address,
        0,
      )
      expect(_roundId).to.be.equal(3)
      await aggregator.contract.connect(oracle0).submit(_roundId, 123)
    }

    {
      // oracle 1
      // skipping round 2, should submit to round 3
      const { _eligibleToSubmit, _roundId } = await aggregator.contract.oracleRoundState(
        oracle1.address,
        0,
      )
      expect(_roundId).to.be.equal(3)
    }
  })
})
