const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { deploy: deployPrepayment } = require('./Prepayment.utils.cjs')
const {
  deploy: deployRrCoordinator,
  parseOracleRegisterdTx,
} = require('./RequestResponseCoordinator.utils.cjs')
const { createSigners } = require('../utils.cjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account1: protocolFeeRecipient,
    account2: consumerSigner,
  } = await createSigners()

  // PREPAYMENT
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)

  // COORDINATOR
  const coordinatorContract = await deployRrCoordinator(prepaymentContract.address, deployerSigner)
  expect(await coordinatorContract.connect(consumerSigner).typeAndVersion()).to.be.equal(
    'RequestResponseCoordinator v0.1',
  )

  return {
    coordinatorContract,
    consumerSigner,
  }
}

describe('RequestResponseCoordinator', function () {
  it('Register oracle', async function () {
    const { coordinatorContract, consumerSigner } = await loadFixture(deploy)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()
    expect(oracle1).to.not.be.equal(oracle2)

    // oracle 1 is not registered yet
    expect(
      await coordinatorContract.connect(consumerSigner).isOracleRegistered(oracle1),
    ).to.be.equal(false)

    // Register oracle 1
    {
      const tx = await (await coordinatorContract.registerOracle(oracle1)).wait()
      const { oracle } = parseOracleRegisterdTx(coordinatorContract, tx)
      expect(oracle).to.be.equal(oracle)
    }

    // oracle 1 is now registered
    expect(
      await coordinatorContract.connect(consumerSigner).isOracleRegistered(oracle1),
    ).to.be.equal(true)

    // Register oracle 2
    {
      const tx = await (await coordinatorContract.registerOracle(oracle2)).wait()
      const { oracle } = parseOracleRegisterdTx(coordinatorContract, tx)
      expect(oracle).to.be.equal(oracle2)
    }
  })

  it('Do not allow to register the same oracle twice', async function () {
    const { coordinatorContract } = await loadFixture(deploy)
    const { address: oracle } = ethers.Wallet.createRandom()

    await (await coordinatorContract.registerOracle(oracle)).wait()
    await expect(coordinatorContract.registerOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'OracleAlreadyRegistered',
    )
  })

  it('Deregister registered oracle', async function () {
    const { coordinatorContract } = await loadFixture(deploy)
    const { address: oracle } = ethers.Wallet.createRandom()

    // Cannot deregister underegistered oracle
    await expect(coordinatorContract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoSuchOracle',
    )

    // Registration
    const txRegisterReceipt = await (await coordinatorContract.registerOracle(oracle)).wait()
    expect(txRegisterReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txRegisterReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')
    expect(registerEvent.args['oracle']).to.be.equal(oracle)

    // Deregistration
    const txDeregisterReceipt = await (await coordinatorContract.deregisterOracle(oracle)).wait()
    expect(txDeregisterReceipt.events.length).to.be.equal(1)
    const deregisterEvent = coordinatorContract.interface.parseLog(txDeregisterReceipt.events[0])
    expect(deregisterEvent.name).to.be.equal('OracleDeregistered')
    expect(deregisterEvent.args['oracle']).to.be.equal(oracle)
  })
})
