const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { vrfConfig } = require('../vrf/VRFCoordinator.config.cjs')
const { parseKlay, getBalance, createSigners } = require('../utils.cjs')
const {
  setupOracle,
  generateVrf,
  deploy: deployVrfCoordinator
} = require('../vrf/VRFCoordinator.utils.cjs')
const { deploy: deployPrepayment, addCoordinator } = require('./Prepayment.utils.cjs')

const {
  deploy: deployRegistry,
  propose,
  confirm,
  setProposeFee,
  withdraw,
  editChainInfor,
  addAggregator,
  removeAggregator,
  createAccount,
  addConsumer,
  removeConsumer
} = require('./Registry.utils.cjs')
const SINGLE_WORD = 1
async function fulfillRandomWords(
  coordinator,
  registeredOracleSigner,
  notRegisteredOracleSigner,
  pi,
  rc,
  isDirectPayment
) {
  // Random word request cannot be fulfilled by an unregistered oracle
  await expect(
    coordinator.connect(notRegisteredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).to.be.revertedWithCustomError(coordinator, 'NoSuchProvingKey')

  // Registered oracle can submit data back to chain
  const tx = await (
    await coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).wait()

  // However even registered oracle cannot fulfill the request more than once
  await expect(
    coordinator.connect(registeredOracleSigner).fulfillRandomWords(pi, rc, isDirectPayment)
  ).to.be.revertedWithCustomError(coordinator, 'NoCorrespondingRequest')

  return tx
}

function validateRandomWordsRequestedEvent(
  tx,
  coordinatorContract,
  keyHash,
  accId,
  maxGasLimit,
  numWords,
  sender,
  isDirectPayment
) {
  let eventIndex = expect(tx.events.length).to.be.equal(5)
  eventIndex = 2

  const event = coordinatorContract.interface.parseLog(tx.events[eventIndex])
  expect(event.name).to.be.equal('RandomWordsRequested')
  const {
    keyHash: eKeyHash,
    requestId,
    preSeed,
    accId: eAccId,
    callbackGasLimit: eCallbackGasLimit,
    numWords: eNumWords,
    sender: eSender,
    isDirectPayment: eIsDirectPayment
  } = event.args
  expect(eKeyHash).to.be.equal(keyHash)
  if (!isDirectPayment) {
    expect(eAccId).to.be.equal(accId)
  }
  expect(eCallbackGasLimit).to.be.equal(maxGasLimit)
  expect(eNumWords).to.be.equal(numWords)
  expect(eSender).to.be.equal(sender)
  expect(eIsDirectPayment).to.be.equal(isDirectPayment)

  const blockHash = tx.blockHash
  const blockNumber = tx.blockNumber

  return { requestId, preSeed, accId: eAccId, blockHash, blockNumber }
}

async function deploy() {
  const {
    account0: deployerSigner,
    account2,
    account3,
    account4: protocolFeeRecipient
  } = await createSigners()

  // Prepayment
  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner
  }

  // VRFCoordinator

  const coordinatorContract = await deployVrfCoordinator(prepaymentContract.address, deployerSigner)
  expect(await coordinatorContract.typeAndVersion()).to.be.equal('VRFCoordinator v0.1')
  const coordinator = {
    contract: coordinatorContract,
    signer: deployerSigner
  }

  // registry

  let registryContract = await ethers.getContractFactory('Registry', {
    signer: deployerSigner
  })
  registryContract = await registryContract.deploy()
  await registryContract.deployed()
  //setup registry

  const fee = parseKlay(1)
  const pChainID = '100001'
  const jsonRpc = 'https://123'
  const L2Endpoint = account2.address
  const { chainID } = await propose(
    registryContract,
    deployerSigner,
    pChainID,
    jsonRpc,
    L2Endpoint,
    fee
  )
  await confirm(registryContract, deployerSigner, chainID)
  const { accId: rAccId } = await createAccount(registryContract, deployerSigner, chainID)
  //add consumer
  await addConsumer(registryContract, deployerSigner, rAccId, deployerSigner.address)

  let endpointContract = await ethers.getContractFactory('L1Endpoint', {
    signer: deployerSigner
  })
  endpointContract = await endpointContract.deploy(
    coordinatorContract.address,
    registryContract.address
  )
  await endpointContract.deployed()

  await endpointContract.addOracle(deployerSigner.address)

  //add endpoint for registry
  await registryContract.setL1Endpoint(endpointContract.address)

  const endpoint = {
    contract: endpointContract,
    signer: deployerSigner
  }

  const registry = {
    contract: registryContract,
    signer: deployerSigner
  }

  return {
    prepayment,
    coordinator,
    endpoint,
    registry,
    account2,
    account3,
    registrAccount: rAccId
  }
}

describe('L1Endpoint', function () {
  it('add and remove oracle', async function () {
    const { endpoint, account2: oracle } = await loadFixture(deploy)

    const txAdd = await (await endpoint.contract.addOracle(oracle.address)).wait()
    expect(txAdd.events.length).to.be.equal(1)
    const eventAdd = endpoint.contract.interface.parseLog(txAdd.events[0])
    expect(eventAdd.name).to.be.equal('OracleAdded')

    const txRemove = await (await endpoint.contract.removeOracle(oracle.address)).wait()
    expect(txRemove.events.length).to.be.equal(1)
    const eventRemove = endpoint.contract.interface.parseLog(txRemove.events[0])
    expect(eventRemove.name).to.be.equal('OracleRemoved')
  })

  it('requestRandomWords', async function () {
    const {
      endpoint,
      coordinator,
      prepayment,
      registry,
      account2: oracle,
      account3: unregisteredOracle,
      registrAccount
    } = await loadFixture(deploy)

    const { maxGasLimit: callbackGasLimit, keyHash } = vrfConfig()

    // Prepare coordinator
    await setupOracle(coordinator.contract, oracle.address)
    await addCoordinator(prepayment.contract, prepayment.signer, coordinator.contract.address)

    //send balance for endpoint contract
    //deposit

    await registry.contract.deposit(registrAccount, { value: parseKlay('1') })
    const accBalance = await registry.contract.getBalance(registrAccount)
    expect(accBalance).to.be.equal(parseKlay('1'))

    // Request random words
    const l2RequestId = 1
    const txRequestRandomWords = await (
      await endpoint.contract.requestRandomWords(
        keyHash,
        callbackGasLimit,
        SINGLE_WORD,
        registrAccount,
        endpoint.signer.address, // consumer
        l2RequestId
      )
    ).wait()
    expect(txRequestRandomWords.events.length).to.be.equal(5)
    const requestEvent = endpoint.contract.interface.parseLog(txRequestRandomWords.events[4])
    expect(requestEvent.name).to.be.equal('RandomWordRequested')
    const numWords = SINGLE_WORD
    const sender = endpoint.contract.address
    const isDirectPayment = true
    const { preSeed, accId, blockHash, blockNumber } = validateRandomWordsRequestedEvent(
      txRequestRandomWords,
      coordinator.contract,
      keyHash,
      0,
      callbackGasLimit,
      numWords,
      sender,
      isDirectPayment
    )
    const { pi, rc } = await generateVrf(
      preSeed,
      blockHash,
      blockNumber,
      accId,
      callbackGasLimit,
      sender,
      numWords
    )

    const txFulfillRandomWords = await fulfillRandomWords(
      coordinator.contract,
      oracle,
      unregisteredOracle,
      pi,
      rc,
      isDirectPayment
    )

    const fulfillEvent = endpoint.contract.interface.parseLog(txFulfillRandomWords.events[0])
    expect(fulfillEvent.name).to.be.equal('RandomWordFulfilled')
    expect(fulfillEvent.args.sender).to.be.equal(endpoint.signer.address)
  })
})
