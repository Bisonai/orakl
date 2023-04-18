import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import crypto from 'crypto'
import { vrfConfig } from './VRF.config'
import { parseKlay } from './utils'
import { Prepayment } from './Prepayment.utils'
import { setMinBalance } from './Coordinator.utils'

const DUMMY_KEY_HASH = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
const NUM_WORDS = 1

function generateDummyPublicProvingKey() {
  const L = 77
  return crypto
    .getRandomValues(new Uint8Array(L))
    .map((a) => {
      return a % 10
    })
    .reduce((acc, v) => acc + v, '')
}

async function deployFixture() {
  const {
    deployer,
    consumer,
    consumer1: sProtocolFeeRecipient,
    consumer2
  } = await hre.getNamedAccounts()

  // Prepayment
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployer
  })
  prepaymentContract = await prepaymentContract.deploy(sProtocolFeeRecipient)
  await prepaymentContract.deployed()

  // VRFCoordinator
  let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
    signer: deployer
  })
  coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  await coordinatorContract.deployed()

  // VRFConsumerMock
  let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
    signer: consumer
  })
  consumerContract = await consumerContract.deploy(coordinatorContract.address)
  await consumerContract.deployed()

  const prepayment = new Prepayment({
    consumerAddress: consumer,
    prepaymentContractAddress: prepaymentContract.address,
    consumerContractAddress: consumerContract.address
  })
  await prepayment.initialize()

  return {
    deployer,
    consumer,
    consumer2,
    prepaymentContract,
    coordinatorContract,
    consumerContract,
    prepayment
  }
}

describe('VRF contract', function () {
  it('Register oracle', async function () {
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

  it('Do not allow to register the same oracle or public proving key twice', async function () {
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

  it('Deregister registered oracle', async function () {
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

  it('requestRandomWords revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract, prepayment } = await loadFixture(deployFixture)

    const { maxGasLimit } = vrfConfig()
    const accId = await prepayment.createAccount()

    await expect(
      consumerContract.requestRandomWords(DUMMY_KEY_HASH, accId, maxGasLimit, NUM_WORDS)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWordsDirect should revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract } = await loadFixture(deployFixture)

    const { maxGasLimit } = vrfConfig()
    const value = parseKlay(1)

    await expect(
      consumerContract.requestRandomWordsDirectPayment(DUMMY_KEY_HASH, maxGasLimit, NUM_WORDS, {
        value
      })
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWords can be called by onlyOwner', async function () {
    const {
      consumerContract,
      consumer2: nonOwnerAddress,
      prepayment
    } = await loadFixture(deployFixture)

    const consumerContractNonOwnerSigner = await ethers.getContractAt(
      'VRFConsumerMock',
      consumerContract.address,
      nonOwnerAddress
    )
    const { maxGasLimit } = vrfConfig()
    const accId = await prepayment.createAccount()

    await expect(
      consumerContractNonOwnerSigner.requestRandomWords(
        DUMMY_KEY_HASH,
        accId,
        maxGasLimit,
        NUM_WORDS
      )
    ).to.be.revertedWithCustomError(consumerContractNonOwnerSigner, 'OnlyOwner')
  })

  it('requestRandomWords should revert with InsufficientPayment error', async function () {
    // VRF is a paid service that requires a payment through a
    // Prepayment smart contract. Every [regular] account has to have at
    // least `sMinBalance` in their account in order to succeed with
    // VRF request.
    const { consumer, coordinatorContract, consumerContract, prepaymentContract, prepayment } =
      await loadFixture(deployFixture)

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
    await prepaymentContract.addCoordinator(coordinatorContract.address)

    await setMinBalance(coordinatorContract, '0.001')

    const accId = await prepayment.createAccount()
    prepayment.addConsumer(consumerContract.address)

    await expect(
      consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, NUM_WORDS)
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')
  })

  // it('Request through [temporary] account & Fulfill', async function () {
  // charge $KLAY to account
  // generate random number through other script
  // create reporter account
  // submit
  // check for returned value
  // })
})
