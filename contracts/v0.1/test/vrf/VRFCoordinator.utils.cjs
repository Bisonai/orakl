const { expect } = require('chai')
const { ethers } = require('hardhat')
const { vrfConfig } = require('./VRFCoordinator.config.cjs')
const { remove0x } = require('../utils.cjs')
const VRF = import('@bisonai/orakl-vrf')

async function deploy(prepaymentAddress, signer) {
  let contract = await ethers.getContractFactory('VRFCoordinator', {
    signer,
  })
  contract = await contract.deploy(prepaymentAddress)
  await contract.deployed()
  return contract
}

async function setupOracle(coordinator, oracleAddress) {
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig, publicProvingKey } = vrfConfig()
  await coordinator.registerOracle(oracleAddress, publicProvingKey)
  await coordinator.setConfig(maxGasLimit, gasAfterPaymentCalculation, Object.values(feeConfig))
}

async function generateVrf(
  preSeed,
  blockHash,
  blockNumber,
  accId,
  callbackGasLimit,
  sender,
  numWords,
) {
  const { sk, pk, pkX, pkY, publicProvingKey, keyHash } = vrfConfig()

  const alpha = remove0x(
    ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [preSeed, blockHash]),
  )

  // Simulate off-chain proof generation
  const { processVrfRequest } = await VRF
  const { proof, uPoint, vComponents } = processVrfRequest(alpha, {
    sk,
    pk,
    pkX,
    pkY,
    keyHash,
  })

  const pi = [publicProvingKey, proof, preSeed, uPoint, vComponents]
  const rc = [blockNumber, accId, callbackGasLimit, numWords, sender]

  return { pi, rc }
}

function parseRandomWordsRequestedTx(coordinator, tx) {
  const event = coordinator.interface.parseLog(tx.events[0])
  const {
    keyHash,
    requestId,
    preSeed,
    accId,
    callbackGasLimit,
    numWords,
    sender,
    isDirectPayment,
  } = event.args
  const blockHash = tx.blockHash
  const blockNumber = tx.blockNumber

  return {
    keyHash,
    requestId,
    preSeed,
    accId,
    callbackGasLimit,
    numWords,
    sender,
    isDirectPayment,
    blockHash,
    blockNumber,
  }
}

function parseRandomWordsFulfilledTx(coordinator, tx) {
  const event = coordinator.interface.parseLog(tx.events[3])
  const blockHash = tx.blockHash
  const blockNumber = tx.blockNumber

  const { requestId, outputSeed, payment, success } = event.args

  return { requestId, outputSeed, payment, success, blockHash, blockNumber }
}

async function fulfillRandomWords(
  coordinator,
  signer,
  preSeed,
  blockHash,
  blockNumber,
  accId,
  callbackGasLimit,
  sender,
  isDirectPayment,
  numWords,
) {
  const { pi, rc } = await generateVrf(
    preSeed,
    blockHash,
    blockNumber,
    accId,
    callbackGasLimit,
    sender,
    numWords,
  )

  const tx = await (
    await coordinator.connect(signer).fulfillRandomWords(pi, rc, isDirectPayment)
  ).wait()

  return tx
}

function parseRequestCanceledTx(coordinator, tx) {
  const event = coordinator.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('RequestCanceled')
  const { requestId } = event.args
  return { requestId }
}

async function computeExactFee(
  coordinatorContract,
  signer,
  reqCount,
  numSubmission,
  callbackGasLimit,
) {
  const serviceFee = await coordinatorContract
    .connect(signer)
    .estimateFee(reqCount, numSubmission, callbackGasLimit)
  const gasPrice = ethers.BigNumber.from(network.config.gasPrice)
  const gasFee = gasPrice.mul(callbackGasLimit)
  return serviceFee.add(gasFee)
}

module.exports = {
  deploy,
  setupOracle,
  fulfillRandomWords,
  generateVrf,
  parseRandomWordsRequestedTx,
  parseRandomWordsFulfilledTx,
  parseRequestCanceledTx,
  computeExactFee,
}
