import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import { vrfConfig } from './VRF.config'
import { parseKlay } from './utils'
import { createAccount } from './Prepayment.utils'

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
