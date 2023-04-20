import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import { createAccount } from './Prepayment.utils'
import { requestResponseConfig } from './RequestResponse.config.ts'
import { parseKlay } from './utils'

const EVENT_ARGS = [
  'requestId',
  'jobId',
  'accId',
  'callbackGasLimit',
  'sender',
  'isDirectPayment',
  'data'
]

async function deployFixture() {
  const {
    deployer,
    consumer,
    rrOracle0,
    account8,
    consumer1: sProtocolFeeRecipient
  } = await hre.getNamedAccounts()
  const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()

  // PREPAYMENT
  let prepaymentContract = await ethers.getContractFactory('Prepayment', {
    signer: deployer
  })
  prepaymentContract = await prepaymentContract.deploy(sProtocolFeeRecipient)
  await prepaymentContract.deployed()

  // COORDINATOR
  let coordinatorContract = await ethers.getContractFactory('RequestResponseCoordinator', {
    signer: deployer
  })
  coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  await coordinatorContract.deployed()

  // COORDINATOR SETTINGS
  await (
    await coordinatorContract.setConfig(maxGasLimit, gasAfterPaymentCalculation, feeConfig)
  ).wait()

  await (await coordinatorContract.registerOracle(rrOracle0)).wait()
  await (await coordinatorContract.registerOracle(account8)).wait()

  const minBalance = ethers.utils.parseUnits('0.001')
  await coordinatorContract.setMinBalance(minBalance)

  // CONNECT COORDINATOR AND PREPAYMENT
  await (await prepaymentContract.addCoordinator(coordinatorContract.address)).wait()

  // CONSUMER
  let consumerContract = await ethers.getContractFactory('RequestResponseConsumerMock', {
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
  return {
    accId,
    maxGasLimit,
    deployer,
    consumer,
    rrOracle0,
    prepaymentContract,
    coordinatorContract,
    consumerContract,
    account8
  }
}

async function consumerDeposit(value: number) {
  const { accId, prepaymentContract, consumer } = await loadFixture(deployFixture)
  // consumer deposit
  const prepaymentContractConsumerSigner = await ethers.getContractAt(
    'Prepayment',
    prepaymentContract.address,
    consumer
  )
  value = parseKlay(value)
  await prepaymentContractConsumerSigner.deposit(accId, { value })
}

describe('Request-Response user contract', function () {
  it('Request & Fulfill Uint256', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      rrOracle0
    } = await loadFixture(deployFixture)
    await consumerDeposit(1)
    const requestReceipt = await (
      await consumerContract.requestDataUint256(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = 123
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false

    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'RequestResponseCoordinator',
      coordinatorContract.address,
      rrOracle0
    )

    const fulfillReceipt = await (
      await coordinatorContractOracleSigner.fulfillDataRequestUint256(
        requestId,
        response,
        requestCommitment,
        isDirectPayment
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)
    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledUint256')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(Number(await consumerContract.sResponse())).to.be.equal(response)
  })

  it('Request & Fulfill Int256', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      rrOracle0,
      account8
    } = await loadFixture(deployFixture)
    await consumerDeposit(1)
    const submissionAmount = 2
    const requestReceipt = await (
      await consumerContract.requestDataInt256(accId, maxGasLimit, submissionAmount, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = 123
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false
    const accounts = [rrOracle0, account8]
    let coordinatorContractOracleSigner: any
    let fulfillReceipt: any
    for (let i = 0; i < submissionAmount; i++) {
      const acc = accounts[i]
      coordinatorContractOracleSigner = await ethers.getContractAt(
        'RequestResponseCoordinator',
        coordinatorContract.address,
        acc
      )

      fulfillReceipt = await (
        await coordinatorContractOracleSigner.fulfillDataRequestInt256(
          requestId,
          response,
          requestCommitment,
          isDirectPayment
        )
      ).wait()
    }

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledInt256')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(Number(await consumerContract.sResponseInt256())).to.be.equal(response)
  })

  it('Request & Fulfill bool', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      rrOracle0
    } = await loadFixture(deployFixture)
    await consumerDeposit(1)
    const requestReceipt = await (
      await consumerContract.requestDataBool(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = true
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false

    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'RequestResponseCoordinator',
      coordinatorContract.address,
      rrOracle0
    )

    const fulfillReceipt = await (
      await coordinatorContractOracleSigner.fulfillDataRequestBool(
        requestId,
        response,
        requestCommitment,
        isDirectPayment
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledBool')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(await consumerContract.sResponseBool()).to.be.equal(response)
  })

  it('Request & Fulfill string', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      rrOracle0
    } = await loadFixture(deployFixture)
    await consumerDeposit(1)
    const requestReceipt = await (
      await consumerContract.requestDataString(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = 'Hello'
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false

    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'RequestResponseCoordinator',
      coordinatorContract.address,
      rrOracle0
    )

    const fulfillReceipt = await (
      await coordinatorContractOracleSigner.fulfillDataRequestString(
        requestId,
        response,
        requestCommitment,
        isDirectPayment
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledString')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(await consumerContract.sResponseString()).to.be.equal(response)
  })

  it('Request & Fulfill Bytes32', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      rrOracle0
    } = await loadFixture(deployFixture)
    await consumerDeposit(1)
    const requestReceipt = await (
      await consumerContract.requestDataBytes32(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = ethers.utils.formatBytes32String('hello')
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false

    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'RequestResponseCoordinator',
      coordinatorContract.address,
      rrOracle0
    )

    const fulfillReceipt = await (
      await coordinatorContractOracleSigner.fulfillDataRequestBytes32(
        requestId,
        response,
        requestCommitment,
        isDirectPayment
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledBytes32')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(await consumerContract.sResponseBytes32()).to.be.equal(response)
  })

  it('Request & Fulfill Bytes', async function () {
    const {
      accId,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      consumer,
      rrOracle0
    } = await loadFixture(deployFixture)
    const prepaymentContractConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      consumer
    )
    const value = parseKlay(1)
    await prepaymentContractConsumerSigner.deposit(accId, { value })
    const requestReceipt = await (
      await consumerContract.requestDataBytes(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    for (const arg of EVENT_ARGS) {
      expect(requestEvent.args[arg]).to.not.be.undefined
    }

    const { requestId } = requestEvent.args

    const response = ethers.utils.formatBytes32String('hello')
    const requestCommitment = {
      blockNum: requestReceipt.blockNumber,
      accId,
      callbackGasLimit: maxGasLimit,
      sender: consumerContract.address
    }
    const isDirectPayment = false

    const coordinatorContractOracleSigner = await ethers.getContractAt(
      'RequestResponseCoordinator',
      coordinatorContract.address,
      rrOracle0
    )
    const fulfillReceipt = await (
      await coordinatorContractOracleSigner.fulfillDataRequestBytes(
        requestId,
        response,
        requestCommitment,
        isDirectPayment
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.greaterThan(1)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(
      fulfillReceipt.events[fulfillReceipt.events.length - 1]
    )
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilledBytes')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(await consumerContract.sResponseBytes()).to.be.equal(response)
  })

  it('requestData should revert with InsufficientPayment error', async function () {
    const { accId, maxGasLimit, consumerContract, coordinatorContract } = await loadFixture(
      deployFixture
    )

    await expect(
      consumerContract.requestDataInt256(accId, maxGasLimit, 1, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')
  })
})
