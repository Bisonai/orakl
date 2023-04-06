import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import crypto from 'crypto'
import { vrfConfig } from './VRF.config'
import { parseKlay } from './utils'
import { createAccount } from './Prepayment.utils'

function generateDummyPublicProvingKey() {
  const L = 77
  return crypto
    .getRandomValues(new Uint8Array(L))
    .map((a) => {
      return a % 10
    })
    .reduce((acc, v) => acc + v, '')
}

describe('VRF contract', function () {
  async function deployFixture() {
    const { deployer, consumer } = await hre.getNamedAccounts()

    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy()
    await prepaymentContract.deployed()

    let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
      signer: deployer
    })
    coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
    await coordinatorContract.deployed()

    //coordinator contract settings
    const minBalance = ethers.utils.parseUnits('0.001')
    await coordinatorContract.setMinBalance(minBalance)

    let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
      signer: consumer
    })
    consumerContract = await consumerContract.deploy(coordinatorContract.address)
    await consumerContract.deployed()

    const accId = await createAccount(
      await coordinatorContract.getPrepaymentAddress(),
      consumerContract.address,
      false,
      true
    )

    const dummyKeyHash = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'

    return {
      accId,
      deployer,
      consumer,
      coordinatorContract,
      consumerContract,
      dummyKeyHash,
      prepaymentContract
    }
  }

  it('should register oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Registration
    const txReceipt = await (
      await coordinatorContract.registerOracle(oracle, publicProvingKey)
    ).wait()

    expect(txReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')

    expect(registerEvent.args['oracle']).to.be.equal(oracle)
    expect(registerEvent.args['keyHash']).to.not.be.undefined
  })

  it('should not allow to register the same oracle or public proving key twice', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle1 } = ethers.Wallet.createRandom()
    const { address: oracle2 } = ethers.Wallet.createRandom()
    const publicProvingKey1 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    const publicProvingKey2 = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]
    expect(oracle1).to.not.be.equal(oracle2)
    expect(publicProvingKey1).to.not.be.equal(publicProvingKey2)

    // Registration
    await (await coordinatorContract.registerOracle(oracle1, publicProvingKey1)).wait()
    // Neither oracle or public proving key can be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle1, publicProvingKey1)
    ).to.be.revertedWithCustomError(coordinatorContract, 'OracleAlreadyRegistered')

    // Oracle cannot be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle1, publicProvingKey2)
    ).to.be.revertedWithCustomError(coordinatorContract, 'OracleAlreadyRegistered')

    // Public proving key cannot be registered twice
    await expect(
      coordinatorContract.registerOracle(oracle2, publicProvingKey1)
    ).to.be.revertedWithCustomError(coordinatorContract, 'ProvingKeyAlreadyRegistered')
  })

  it('should allow to deregister registered oracle', async function () {
    const { coordinatorContract } = await loadFixture(deployFixture)
    const { address: oracle } = ethers.Wallet.createRandom()
    const publicProvingKey = [generateDummyPublicProvingKey(), generateDummyPublicProvingKey()]

    // Cannot deregister underegistered oracle
    await expect(coordinatorContract.deregisterOracle(oracle)).to.be.revertedWithCustomError(
      coordinatorContract,
      'NoSuchOracle'
    )

    // Registration
    const txRegisterReceipt = await (
      await coordinatorContract.registerOracle(oracle, publicProvingKey)
    ).wait()
    expect(txRegisterReceipt.events.length).to.be.equal(1)
    const registerEvent = coordinatorContract.interface.parseLog(txRegisterReceipt.events[0])
    expect(registerEvent.name).to.be.equal('OracleRegistered')
    const kh = registerEvent.args['keyHash']
    expect(kh).to.not.be.undefined

    // Deregistration
    const txDeregisterReceipt = await (await coordinatorContract.deregisterOracle(oracle)).wait()
    expect(txDeregisterReceipt.events.length).to.be.equal(1)
    const deregisterEvent = coordinatorContract.interface.parseLog(txDeregisterReceipt.events[0])
    expect(deregisterEvent.name).to.be.equal('OracleDeregistered')
    expect(deregisterEvent.args['oracle']).to.be.equal(oracle)
    expect(deregisterEvent.args['keyHash']).to.be.equal(kh)
  })

  it('requestRandomWords should revert on InvalidKeyHash', async function () {
    const { accId, coordinatorContract, consumerContract, dummyKeyHash } = await loadFixture(
      deployFixture
    )

    const { maxGasLimit } = vrfConfig()
    const numWords = 1

    await expect(
      consumerContract.requestRandomWords(dummyKeyHash, accId, maxGasLimit, numWords)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWordsDirect should revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract, dummyKeyHash } = await loadFixture(deployFixture)

    const { maxGasLimit } = vrfConfig()
    const numWords = 1
    const value = parseKlay(1)

    await expect(
      consumerContract.requestRandomWordsDirect(dummyKeyHash, maxGasLimit, numWords, { value })
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWords should revert with InsufficientPayment error', async function () {
    const {
      accId,
      consumer,
      coordinatorContract,
      consumerContract,
      dummyKeyHash,
      prepaymentContract
    } = await loadFixture(deployFixture)
    const {
      oracle,
      publicProvingKey,
      keyHash,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    } = vrfConfig()

    await coordinatorContract.registerOracle(oracle, publicProvingKey)

    await coordinatorContract.setConfig(
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    const prepaymentContractConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      consumer
    )

    await prepaymentContractConsumerSigner.addConsumer(accId, consumerContract.address)
    await prepaymentContract.addCoordinator(coordinatorContract.address)
    const numWords = 1

    await expect(
      consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, numWords)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')
  })
})
