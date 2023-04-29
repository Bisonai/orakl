const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const {
  setupOracle: setupVrfCoordinator,
  parseRandomWordsRequestedTx,
  fulfillRandomWords,
  parseRandomWordsFulfilledTx
} = require('./VRFCoordinator.utils.cjs')

const {
  setupOracle: setupRequestResponseCoordinator,
  parseDataRequestFulfilledTx
} = require('./RequestResponseCoordinator.utils.cjs')
const { createAccount, deposit } = require('./Prepayment.utils.cjs')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { requestResponseConfig } = require('./RequestResponse.config.cjs')
const { getBalance } = require('./utils.cjs')
const oraklVrf = import('@bisonai/orakl-vrf')

async function createSigners() {
  let { deployer, consumer, consumer1, vrfOracle0, rrOracle0 } = await hre.getNamedAccounts()

  const deployerSigner = await ethers.getSigner(deployer)
  const consumerSigner = await ethers.getSigner(consumer)
  const protocolFeeRecipientSigner = await ethers.getSigner(consumer1)
  const vrfOracleSigner = await ethers.getSigner(vrfOracle0)
  const rrOracleSigner = await ethers.getSigner(rrOracle0)

  return {
    deployerSigner,
    consumerSigner,
    protocolFeeRecipientSigner,
    vrfOracleSigner,
    rrOracleSigner
  }
}

async function deploy() {
  const {
    deployerSigner,
    consumerSigner,
    protocolFeeRecipientSigner,
    vrfOracleSigner,
    rrOracleSigner
  } = await createSigners()

  // Prepayment
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployerSigner
  })
  prepaymentContract = await prepaymentContract.deploy(protocolFeeRecipientSigner.address)
  await prepaymentContract.deployed()

  // VRFCoordinator
  let vrfCoordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
    signer: deployerSigner
  })
  vrfCoordinatorContract = await vrfCoordinatorContract.deploy(prepaymentContract.address)
  await vrfCoordinatorContract.deployed()

  // VRFCoordinator setup
  await setupVrfCoordinator(vrfCoordinatorContract, vrfOracleSigner.address)
  await prepaymentContract.addCoordinator(vrfCoordinatorContract.address)

  // RequestResponseCoordinator
  let rrCoordinatorContract = await ethers.getContractFactory('RequestResponseCoordinator', {
    signer: deployerSigner
  })
  rrCoordinatorContract = await rrCoordinatorContract.deploy(prepaymentContract.address)
  await rrCoordinatorContract.deployed()

  // RequestResponseCoordinator setup
  await setupRequestResponseCoordinator(rrCoordinatorContract, rrOracleSigner.address)
  await prepaymentContract.addCoordinator(rrCoordinatorContract.address)

  // VRFConsumerRevertFulfillmentMock
  let vrfConsumerContract = await ethers.getContractFactory('VRFConsumerRevertFulfillmentMock', {
    signer: consumerSigner
  })
  vrfConsumerContract = await vrfConsumerContract.deploy(vrfCoordinatorContract.address)
  await vrfConsumerContract.deployed()

  // RequestResponseConsumerRevertFulfillmentMock
  let rrConsumerContract = await ethers.getContractFactory(
    'RequestResponseConsumerRevertFulfillmentMock',
    {
      signer: consumerSigner
    }
  )
  rrConsumerContract = await rrConsumerContract.deploy(rrCoordinatorContract.address)
  await rrConsumerContract.deployed()

  const { accId } = await createAccount(prepaymentContract, consumerSigner)
  await deposit(prepaymentContract, consumerSigner, accId, '1')
  await prepaymentContract.connect(consumerSigner).addConsumer(accId, vrfConsumerContract.address)
  await prepaymentContract.connect(consumerSigner).addConsumer(accId, rrConsumerContract.address)

  return {
    vrfCoordinatorContract,
    rrCoordinatorContract,
    vrfConsumerContract,
    rrConsumerContract,
    accId
  }
}

describe('Revert Fulfillment Test', function () {
  it('Revert VRF', async function () {
    const { vrfCoordinatorContract, vrfConsumerContract, accId } = await loadFixture(deploy)
    const { vrfOracleSigner, protocolFeeRecipientSigner } = await createSigners()

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const numWords = 1
    const txRequest = await (
      await vrfConsumerContract.requestRandomWords(keyHash, accId, callbackGasLimit, numWords)
    ).wait()

    const { preSeed, sender, isDirectPayment, blockHash, blockNumber } =
      parseRandomWordsRequestedTx(vrfCoordinatorContract, txRequest)

    const protocolFeeRecipientBalanceBefore = await getBalance(protocolFeeRecipientSigner.address)
    const oracleBalanceBefore = await getBalance(vrfOracleSigner.address)

    const txFulfill = await fulfillRandomWords(
      vrfCoordinatorContract,
      vrfOracleSigner,
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      isDirectPayment,
      numWords
    )

    const { payment, success } = parseRandomWordsFulfilledTx(vrfCoordinatorContract, txFulfill)
    expect(payment).to.be.above(0)
    expect(success).to.be.equal(false)

    const protocolFeeRecipientBalanceAfter = await getBalance(protocolFeeRecipientSigner.address)
    expect(protocolFeeRecipientBalanceAfter).to.be.gt(protocolFeeRecipientBalanceBefore)

    const oracleBalanceAfter = await getBalance(vrfOracleSigner.address)
    expect(oracleBalanceAfter).to.be.gt(oracleBalanceBefore)
  })

  it('Revert Fulfillment Test', async function () {
    const { rrCoordinatorContract, rrConsumerContract, accId } = await loadFixture(deploy)
    const { rrOracleSigner, protocolFeeRecipientSigner } = await createSigners()

    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    const numSubmission = 1

    const tx = await (
      await rrConsumerContract.requestDataUint128(accId, callbackGasLimit, numSubmission)
    ).wait()
    const { requestId, sender, blockNumber } = parseRandomWordsRequestedTx(
      rrCoordinatorContract,
      tx
    )

    const protocolFeeRecipientBalanceBefore = await getBalance(protocolFeeRecipientSigner.address)
    const oracleBalanceBefore = await getBalance(rrOracleSigner.address)

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender
    }
    const isDirectPayment = false
    const txFulfill = await (
      await rrCoordinatorContract
        .connect(rrOracleSigner)
        .fulfillDataRequestInt256(requestId, 123, requestCommitment, isDirectPayment)
    ).wait()

    const { payment, success } = parseDataRequestFulfilledTx(
      rrCoordinatorContract,
      txFulfill,
      'DataRequestFulfilledInt256'
    )
    expect(payment).to.be.above(0)
    expect(success).to.be.equal(false)

    const protocolFeeRecipientBalanceAfter = await getBalance(protocolFeeRecipientSigner.address)
    expect(protocolFeeRecipientBalanceAfter).to.be.gt(protocolFeeRecipientBalanceBefore)

    const oracleBalanceAfter = await getBalance(rrOracleSigner.address)
    expect(oracleBalanceAfter).to.be.gt(oracleBalanceBefore)
  })
})
