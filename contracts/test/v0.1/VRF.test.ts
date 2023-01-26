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

    let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
      signer: deployer
    })
    coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)

    let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
      signer: consumer
    })
    consumerContract = await consumerContract.deploy(coordinatorContract.address)

    const accId = await createAccount(prepaymentContract)

    const dummyKeyHash = '0x00000773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'

    return {
      accId,
      deployer,
      consumer,
      prepaymentContract,
      coordinatorContract,
      consumerContract,
      dummyKeyHash
    }
  }

  it('requestRandomWords should revert on InvalidKeyHash', async function () {
    const { accId, coordinatorContract, consumerContract, dummyKeyHash } = await loadFixture(
      deployFixture
    )

    const { minimumRequestConfirmations, maxGasLimit } = vrfConfig()
    const numWords = 1

    await expect(
      consumerContract.requestRandomWords(
        dummyKeyHash,
        accId,
        minimumRequestConfirmations,
        maxGasLimit,
        numWords
      )
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })

  it('requestRandomWordsDirect should revert on InvalidKeyHash', async function () {
    const { coordinatorContract, consumerContract, dummyKeyHash } = await loadFixture(deployFixture)

    const { minimumRequestConfirmations, maxGasLimit } = vrfConfig()
    const numWords = 1
    const value = parseKlay(1)

    await expect(
      consumerContract.requestRandomWordsDirect(
        dummyKeyHash,
        minimumRequestConfirmations,
        maxGasLimit,
        numWords,
        { value }
      )
    ).to.be.revertedWithCustomError(coordinatorContract, 'InvalidKeyHash')
  })
})
