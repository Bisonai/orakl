import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'

async function deployFixture() {
  const {
    deployer,
    consumer,
    rrOracle0,
    consumer1: sProtocolFeeRecipient
  } = await hre.getNamedAccounts()

  // PREPAYMENT
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployer
  })
  prepaymentContract = await prepaymentContract.deploy(sProtocolFeeRecipient)
  await prepaymentContract.deployed()

  // COORDINATOR
  let coordinatorContract = await ethers.getContractFactory('RequestResponseCoordinator', {
    signer: deployer
  })
  coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  await coordinatorContract.deployed()

  return {
    deployer,
    coordinatorContract
  }
}

describe('RequestResponseCoordinator', function () {
  it('Register oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()
    expect(oracle1).to.not.be.equal(oracle2)

    // Register oracle 1
    const txReceipt = await (await coordinatorContract.registerOracle(oracle1)).wait()
    expect(txReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')
    expect(registerEvent.args['oracle']).to.be.equal(oracle1)

    // Register oracle 2
    await (await coordinatorContract.registerOracle(oracle2)).wait()
  })

  it('Do not allow to register the same oracle twice', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()

    await (await coordinatorContract.registerOracle(oracle)).wait()
    await expect(coordinatorContract.registerOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'OracleAlreadyRegistered'
    )
  })

  it('Deregister registered oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()

    // Cannot deregister underegistered oracle
    await expect(coordinatorContract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoSuchOracle'
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
