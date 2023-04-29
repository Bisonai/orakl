const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { deploy: deployVrfConsumerMock } = require('./VRFConsumerMock.utils.cjs')
const {
  deploy: deployVrfCoordinator,
  setupOracle: setupVrfOracle,
  parseRandomWordsRequestedTx
} = require('./VRFCoordinator.utils.cjs')
const { createAccount, addConsumer, deposit } = require('./Prepayment.utils.cjs')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { parseRequestCanceled } = require('./CoordinatorBase.utils.cjs')
const { deploy: deployPrepayment } = require('./Prepayment.utils.cjs')

async function createSigners() {
  let { deployer, consumer, consumer1, vrfOracle0 } = await hre.getNamedAccounts()

  const deployerSigner = await ethers.getSigner(deployer)
  const consumerSigner = await ethers.getSigner(consumer)
  const invalidConsumerSigner = await ethers.getSigner(consumer1)
  const vrfOracleSigner = await ethers.getSigner(vrfOracle0)
  const protocolFeeRecipientSigner = await ethers.getSigner(consumer1)

  return {
    deployerSigner,
    consumerSigner,
    invalidConsumerSigner,
    vrfOracleSigner,
    protocolFeeRecipientSigner
  }
}

async function deploy() {
  const {
    deployerSigner,
    consumerSigner,
    vrfOracleSigner,
    protocolFeeRecipientSigner,
    invalidConsumerSigner
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner
  )

  // VRFCoordinator
  const vrfCoordinatorContract = await deployVrfCoordinator(
    prepaymentContract.address,
    deployerSigner
  )

  // VRFConsumerMock
  const consumerContract = await deployVrfConsumerMock(
    vrfCoordinatorContract.address,
    consumerSigner
  )

  return {
    vrfOracleSigner,
    consumerSigner,
    prepaymentContract,
    vrfCoordinatorContract,
    consumerContract,
    invalidConsumerSigner
  }
}

describe('CoordinatorBase', function () {
  it('VRF: Cannot cancel an invalid request', async function () {
    const { prepaymentContract, consumerContract, vrfCoordinatorContract, consumerSigner } =
      await loadFixture(deploy)

    await setupVrfOracle(vrfCoordinatorContract, consumerContract.address)
    await prepaymentContract.addCoordinator(vrfCoordinatorContract.address)

    const invalidRequestId = 123
    await expect(
      vrfCoordinatorContract.connect(consumerSigner).cancelRequest(invalidRequestId)
    ).to.be.revertedWithCustomError(vrfCoordinatorContract, 'NoCorrespondingRequest')
  })

  it('VRF: Request can be canceled by request initiator only', async function () {
    const {
      prepaymentContract,
      vrfCoordinatorContract,
      consumerContract,
      consumerSigner,
      invalidConsumerSigner
    } = await loadFixture(deploy)

    // oracle setup
    await setupVrfOracle(vrfCoordinatorContract, consumerContract.address)
    await prepaymentContract.addCoordinator(vrfCoordinatorContract.address)

    // account setup
    const { accId } = await createAccount(prepaymentContract, consumerSigner)
    await addConsumer(prepaymentContract, consumerSigner, accId, consumerContract.address)
    await deposit(prepaymentContract, consumerSigner, accId, '2')

    // request
    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const numWords = 1
    const txRequest = await (
      await consumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, numWords)
    ).wait()
    const { requestId, sender } = parseRandomWordsRequestedTx(vrfCoordinatorContract, txRequest)

    // cancel with wrong signer
    await expect(
      vrfCoordinatorContract.connect(invalidConsumerSigner).cancelRequest(requestId)
    ).to.be.revertedWithCustomError(vrfCoordinatorContract, 'NotRequestOwner')

    // cancel with right signer
    const txCancel = await (await consumerContract.cancelRequest(requestId)).wait()
    parseRequestCanceled(vrfCoordinatorContract, txCancel)
  })
})
