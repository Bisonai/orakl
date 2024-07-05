const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const {
  deploy: deployVrfCoordinator,
  setupOracle: setupVrfCoordinator,
  parseRandomWordsRequestedTx,
  fulfillRandomWords,
  parseRandomWordsFulfilledTx,
} = require('./VRFCoordinator.utils.cjs')
const { parseKlay } = require('../utils.cjs')
const {
  deploy: deployRrCoordinator,
  setupOracle: setupRequestResponseCoordinator,
  parseDataRequestedTx,
  parseDataRequestFulfilledTx,
} = require('../non-vrf/RequestResponseCoordinator.utils.cjs')
const {
  deploy: deployPrepayment,
  createAccount,
  deposit,
} = require('../non-vrf/Prepayment.utils.cjs')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { requestResponseConfig } = require('../non-vrf/RequestResponse.config.cjs')
const { getBalance, createSigners } = require('../utils.cjs')
const oraklVrf = import('@bisonai/orakl-vrf')

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2: vrfOracleSigner,
    account3: rrOracleSigner,
    account4: protocolFeeRecipientSigner,
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner,
  )
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner,
  }

  // VRFCoordinator
  const vrfCoordinatorContract = await deployVrfCoordinator(
    prepaymentContract.address,
    deployerSigner,
  )
  const vrfCoordinator = {
    contract: vrfCoordinatorContract,
    signer: deployerSigner,
  }

  // VRFCoordinator setup
  await setupVrfCoordinator(vrfCoordinatorContract, vrfOracleSigner.address)
  await prepaymentContract.addCoordinator(vrfCoordinatorContract.address)

  // RequestResponseCoordinator
  const rrCoordinatorContract = await deployRrCoordinator(
    prepaymentContract.address,
    deployerSigner,
  )
  const rrCoordinator = {
    contract: rrCoordinatorContract,
    signer: deployerSigner,
  }

  // RequestResponseCoordinator setup
  await setupRequestResponseCoordinator(rrCoordinatorContract, rrOracleSigner.address)
  await prepaymentContract.addCoordinator(rrCoordinatorContract.address)

  // VRFConsumerRevertFulfillmentMock
  let vrfConsumerContract = await ethers.getContractFactory('VRFConsumerRevertFulfillmentMock', {
    signer: consumerSigner,
  })
  vrfConsumerContract = await vrfConsumerContract.deploy(vrfCoordinatorContract.address)
  await vrfConsumerContract.deployed()
  const vrfConsumer = {
    contract: vrfConsumerContract,
    signer: consumerSigner,
  }

  // RequestResponseConsumerRevertFulfillmentMock
  let rrConsumerContract = await ethers.getContractFactory(
    'RequestResponseConsumerRevertFulfillmentMock',
    {
      signer: consumerSigner,
    },
  )
  rrConsumerContract = await rrConsumerContract.deploy(rrCoordinatorContract.address)
  await rrConsumerContract.deployed()
  const rrConsumer = {
    contract: rrConsumerContract,
    signer: consumerSigner,
  }

  const { accId } = await createAccount(prepaymentContract, consumerSigner)
  const amount = parseKlay(1)
  await deposit(prepaymentContract, consumerSigner, accId, amount)
  await prepaymentContract.connect(consumerSigner).addConsumer(accId, vrfConsumerContract.address)
  await prepaymentContract.connect(consumerSigner).addConsumer(accId, rrConsumerContract.address)

  return {
    vrfOracleSigner,
    rrOracleSigner,
    protocolFeeRecipientSigner,
    vrfCoordinator,
    vrfConsumer,
    rrCoordinator,
    rrConsumer,
    accId,
  }
}

describe('Revert Fulfillment Test', function () {
  it('Revert VRF', async function () {
    const { vrfOracleSigner, protocolFeeRecipientSigner, vrfCoordinator, vrfConsumer, accId } =
      await loadFixture(deploy)

    const { keyHash, maxGasLimit: callbackGasLimit } = vrfConfig()
    const numWords = 1
    const txRequest = await (
      await vrfConsumer.contract.requestRandomWords(keyHash, accId, callbackGasLimit, numWords)
    ).wait()

    const { preSeed, sender, isDirectPayment, blockHash, blockNumber } =
      parseRandomWordsRequestedTx(vrfCoordinator.contract, txRequest)

    const protocolFeeRecipientBalanceBefore = await getBalance(protocolFeeRecipientSigner.address)
    const oracleBalanceBefore = await getBalance(vrfOracleSigner.address)

    const txFulfill = await fulfillRandomWords(
      vrfCoordinator.contract,
      vrfOracleSigner,
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      isDirectPayment,
      numWords,
    )

    const { payment, success } = parseRandomWordsFulfilledTx(vrfCoordinator.contract, txFulfill)
    expect(payment).to.be.above(0)
    expect(success).to.be.equal(false)

    const protocolFeeRecipientBalanceAfter = await getBalance(protocolFeeRecipientSigner.address)
    expect(protocolFeeRecipientBalanceAfter).to.be.gt(protocolFeeRecipientBalanceBefore)

    const protocolRecipientRevenue = protocolFeeRecipientBalanceAfter.sub(
      protocolFeeRecipientBalanceBefore,
    )
    const protocolFee = ethers.BigNumber.from('500000000000000')
    expect(protocolRecipientRevenue).to.be.equal(protocolFee)

    const oracleBalanceAfter = await getBalance(vrfOracleSigner.address)
    // VRF oracle should receive service fee and gas fee rebate
    // after fulfilling callback function even though it reverted.
    expect(oracleBalanceAfter).to.be.gt(oracleBalanceBefore)

    const oracleRevenue = oracleBalanceAfter.sub(oracleBalanceBefore)
    const oracleServiceFee = ethers.BigNumber.from('4500000000000000')
    const extraGasRebate = oracleRevenue.sub(oracleServiceFee)
    expect(extraGasRebate).to.be.gte(0)
    console.log(
      'extraGasRebate',
      extraGasRebate.div(hre.network.config.gasPrice.toString()).toString(),
    )
  })

  it('Revert Request-Response', async function () {
    const { rrOracleSigner, protocolFeeRecipientSigner, rrCoordinator, rrConsumer, accId } =
      await loadFixture(deploy)
    const { maxGasLimit: callbackGasLimit } = requestResponseConfig()
    const numSubmission = 1

    const tx = await (
      await rrConsumer.contract.requestDataUint128(accId, callbackGasLimit, numSubmission)
    ).wait()
    const { requestId, sender, blockNumber, jobId, isDirectPayment } = parseDataRequestedTx(
      rrCoordinator.contract,
      tx,
    )

    const protocolFeeRecipientBalanceBefore = await getBalance(protocolFeeRecipientSigner.address)
    const oracleBalanceBefore = await getBalance(rrOracleSigner.address)

    const requestCommitment = {
      blockNum: blockNumber,
      accId,
      callbackGasLimit,
      numSubmission,
      sender,
      isDirectPayment,
      jobId,
    }

    const txFulfill = await (
      await rrCoordinator.contract
        .connect(rrOracleSigner)
        .fulfillDataRequestUint128(requestId, 123, requestCommitment)
    ).wait()

    const { payment, success } = parseDataRequestFulfilledTx(
      rrCoordinator.contract,
      txFulfill,
      'DataRequestFulfilledUint128',
    )
    expect(payment).to.be.above(0)
    expect(success).to.be.equal(false)

    const protocolFeeRecipientBalanceAfter = await getBalance(protocolFeeRecipientSigner.address)
    expect(protocolFeeRecipientBalanceAfter).to.be.gt(protocolFeeRecipientBalanceBefore)

    const protocolRecipientRevenue = protocolFeeRecipientBalanceAfter.sub(
      protocolFeeRecipientBalanceBefore,
    )
    const protocolFee = ethers.BigNumber.from('500000000000000')
    expect(protocolRecipientRevenue).to.be.equal(protocolFee)

    const oracleBalanceAfter = await getBalance(rrOracleSigner.address)
    // Request-Response oracle should receive service fee and gas fee rebate
    // after fulfilling callback function even though it reverted.
    expect(oracleBalanceAfter).to.be.gt(oracleBalanceBefore)

    const oracleRevenue = oracleBalanceAfter.sub(oracleBalanceBefore)
    const oracleServiceFee = ethers.BigNumber.from('4500000000000000')
    const extraGasRebate = oracleRevenue.sub(oracleServiceFee)
    expect(extraGasRebate).to.be.gte(0)
    console.log(
      'extraGasRebate',
      extraGasRebate.div(hre.network.config.gasPrice.toString()).toString(),
    )
  })
})
