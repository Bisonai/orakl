import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'

describe('RequestResponseCoordinator', function () {
  async function deployFixture() {
    const { deployer, consumer, rrOracle0 } = await hre.getNamedAccounts()

    // PREPAYMENT
    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy()
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

  it('should register oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()

    // Register oracle 1
    const txReceipt = await (await coordinatorContract.registerOracle(oracle1)).wait()
    expect(txReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')
    expect(registerEvent.args['oracle']).to.be.equal(oracle1)

    // Register oracle 2
    await (await coordinatorContract.registerOracle(oracle2)).wait()
  })

  it('should not allow to register the same oracle twice', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()

    await (await coordinatorContract.registerOracle(oracle)).wait()
    await expect(coordinatorContract.registerOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'OracleAlreadyRegistered'
    )
  })
})
