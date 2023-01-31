import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { ethers } from 'hardhat'
import { createAccount } from './Prepayment.utils'
import { requestResponseConfig } from './RequestResponse.config.ts'

describe('Request-Response user contract', function () {
  async function deployFixture() {
    const { deployer, consumer, rrOracle0 } = await hre.getNamedAccounts()
    const { minimumRequestConfirmations, maxGasLimit, gasAfterPaymentCalculation, feeConfig } =
      requestResponseConfig()

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
      await coordinatorContract.setConfig(
        minimumRequestConfirmations,
        maxGasLimit,
        gasAfterPaymentCalculation,
        feeConfig
      )
    ).wait()

    await (await coordinatorContract.registerOracle(rrOracle0)).wait()

    // CONNECT COORDINATOR AND PREPAYMENT
    await (await prepaymentContract.addCoordinator(coordinatorContract.address)).wait()

    // CONSUMER
    let consumerContract = await ethers.getContractFactory('RequestResponseConsumerMock', {
      signer: consumer
    })
    consumerContract = await consumerContract.deploy(coordinatorContract.address)
    await consumerContract.deployed()

    const accId = await createAccount(prepaymentContract.address, consumerContract.address)

    return {
      accId,
      minimumRequestConfirmations,
      maxGasLimit,
      deployer,
      consumer,
      rrOracle0,
      prepaymentContract,
      coordinatorContract,
      consumerContract
    }
  }

  // beforeEach(async function () {
  //     requestResponseCoordinator = await ethers.getContractFactory('RequestResponseCoordinator')
  // requestResponseCoordinator = await requestResponseCoordinator.deploy()
  // await requestResponseCoordinator.deployed()
  //
  // userContract = await ethers.getContractFactory('RequestResponseConsumerMock')
  // userContract = await userContract.deploy(requestResponseCoordinator.address)
  // await userContract.deployed()
  //   })

  it('Request & Fulfill', async function () {
    const {
      accId,
      minimumRequestConfirmations,
      maxGasLimit,
      consumerContract,
      coordinatorContract,
      prepaymentContract,
      consumer,
      rrOracle0
    } = await loadFixture(deployFixture)

    const requestReceipt = await (
      await consumerContract.requestData(accId, minimumRequestConfirmations, maxGasLimit, {
        gasLimit: 500_000
      })
    ).wait()

    // expect(requestReceipt.events.length).to.be.equal(1)
    // const requestEvent = coordinatorContract.interface.parseLog(requestReceipt.events[0])
    // expect(requestEvent.name).to.be.equal('Requested')
    //
    // const eventArgs = [
    //   'requestId',
    //   'jobId',
    //   'accId',
    //   'minimumRequestConfirmations',
    //   'callbackGasLimit',
    //   'sender',
    //   'data'
    // ]
    // for (const arg of eventArgs) {
    //   expect(requestEvent.args[arg]).to.not.be.undefined
    // }
    //
    // const { requestId } = requestEvent.args
    //
    // const response = 123
    // const requestCommitment = {
    //   blockNum: requestReceipt.blockNumber,
    //   accId,
    //   callbackGasLimit: maxGasLimit,
    //   sender: consumerContract.address
    // }
    // const isDirectPayment = false
    //
    // const coordinatorContractOracleSigner = await ethers.getContractAt(
    //   'RequestResponseCoordinator',
    //   coordinatorContract.address,
    //   rrOracle0
    // )
    //
    // const fulfillReceipt = await (
    //   await coordinatorContractOracleSigner.fulfillRequest(
    //     requestId,
    //     response,
    //     requestCommitment,
    //     isDirectPayment,
    //     {
    //       gasLimit: maxGasLimit + 300_000
    //     }
    //   )
    // ).wait()
    //
    // expect(fulfillReceipt.events.length).to.be.equal(2)
    //
    // // PREPAYMENT EVENT
    // const prepaymentEvent = prepaymentContract.interface.parseLog(fulfillReceipt.events[0])
    // expect(prepaymentEvent.name).to.be.equal('AccountBalanceDecreased')
    // expect(prepaymentEvent.args.accId).to.be.equal(accId)
    // // FIXME
    // // expect(prepaymentEvent.args.oldBalance).to.be.equal()
    // // expect(prepaymentEvent.args.newBalance).to.be.equal()
    //
    // // COORDINATOR EVENT
    // const fulfillEvent = coordinatorContract.interface.parseLog(fulfillReceipt.events[1])
    // expect(fulfillEvent.name).to.be.equal('Fulfilled')
    // expect(fulfillEvent.args.requestId).to.be.equal(requestId)
    // expect(Number(await consumerContract.s_response())).to.be.equal(response)
  })
})
