import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import { createAccount } from './Prepayment.utils'
import { requestResponseConfig } from './RequestResponse.config.ts'
import { parseKlay } from './utils'

describe('Request-Response user contract', function () {
  async function deployFixture() {
    const { deployer, consumer, rrOracle0 } = await hre.getNamedAccounts()
    const { maxGasLimit, gasAfterPaymentCalculation, feeConfig } = requestResponseConfig()

    // PREPAYMENT
    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy()
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

    const minBalance = ethers.utils.parseUnits('1', 15)
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
      consumerContract
    }
  }

  it('Request & Fulfill', async function () {
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
      await consumerContract.requestData(accId, maxGasLimit, {
        gasLimit: 500_000
      })
    ).wait()

    expect(requestReceipt.events.length).to.be.equal(1)
    const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    expect(requestEvent.name).to.be.equal('DataRequested')

    const eventArgs = [
      'requestId',
      'jobId',
      'accId',
      'callbackGasLimit',
      'sender',
      'isDirectPayment',
      'data'
    ]
    for (const arg of eventArgs) {
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
      await coordinatorContractOracleSigner.fulfillDataRequest(
        requestId,
        response,
        requestCommitment,
        isDirectPayment,
        {
          gasLimit: maxGasLimit + 300_000
        }
      )
    ).wait()

    expect(fulfillReceipt.events.length).to.be.equal(2)

    // PREPAYMENT EVENT
    const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    expect(prepaymentEvent.args.accId).to.be.equal(accId)

    // FIXME
    // expect(prepaymentEvent.args.oldBalance).to.be.equal()
    // expect(prepaymentEvent.args.newBalance).to.be.equal()

    // COORDINATOR EVENT
    const fulfillEvent = coordinatorContract.interface.parseLog(fulfillReceipt.events[1])
    expect(fulfillEvent.name).to.be.equal('DataRequestFulfilled')
    expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    expect(Number(await consumerContract.s_response())).to.be.equal(response)
  })
  it('requestData should revert with InsufficientPayment error', async function () {
    const { accId, maxGasLimit, consumerContract, coordinatorContract } = await loadFixture(
      deployFixture
    )

    await expect(
      consumerContract.requestData(accId, maxGasLimit, {
        gasLimit: 500_000
      })
    ).to.be.revertedWithCustomError(coordinatorContract, 'InsufficientPayment')
  })
})
