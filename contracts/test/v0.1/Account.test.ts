import { expect } from 'chai'
import { ethers } from 'hardhat'
import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'

describe('Account', function () {
  async function deployPrepayment() {
    const { deployer, consumer, consumer1 } = await hre.getNamedAccounts()

    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy()
    await prepaymentContract.deployed()

    const prepaymentContractConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      consumer
    )

    return { deployer, consumer, consumer1, prepaymentContract, prepaymentContractConsumerSigner }
  }

  // async function deployFixture() {
  //   const { deployer, consumer } = await hre.getNamedAccounts()
  //
  //   let prepaymentContract = await ethers.getContractFactory('Prepayment', {
  //     signer: deployer
  //   })
  //   prepaymentContract = await prepaymentContract.deploy()
  //   await prepaymentContract.deployed()

  // let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
  //   signer: deployer
  // })
  // coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)
  //
  // let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
  //   signer: consumer
  // })
  // consumerContract = await consumerContract.deploy(coordinatorContract.address)
  // await consumerContract.deployed()
  //
  // const accId = await createAccount(prepaymentContract.address, consumerContract.address)
  //
  // const prepaymentContractConsumerSigner = await ethers.getContractAt(
  //   'Prepayment',
  //   prepaymentContract.address,
  //   consumer
  // )
  //
  // return {
  //   accId,
  //   deployer,
  //   consumer,
  //   prepaymentContract,
  //   prepaymentContractConsumerSigner,
  //   coordinatorContract,
  //   consumerContract
  // }
  // }

  it('Create & cancel account', async function () {
    const { prepaymentContractConsumerSigner, consumer, consumer1 } = await loadFixture(
      deployPrepayment
    )

    // Create account ///////////////////////////////////////////////////////////
    const txReceipt = await (await prepaymentContractConsumerSigner.createAccount()).wait()

    expect(txReceipt.events.length).to.be.equal(1)

    const accountCreatedEvent = prepaymentContractConsumerSigner.interface.parseLog(
      txReceipt.events[0]
    )
    expect(accountCreatedEvent.name).to.be.equal('AccountCreated')
    const { accId: id, account, owner } = accountCreatedEvent.args

    expect(owner).to.be.equal(consumer)

    // Access account metadata directly through deployed contract
    const accountContract = await ethers.getContractAt('Account', account, consumer)
    const accountOwner = await accountContract.getOwner()
    expect(owner).to.be.equal(accountOwner)

    const accountId = await accountContract.getAccountId()
    expect(id).to.be.equal(accountId)

    const balance = await accountContract.getBalance()
    expect(balance).to.be.equal(0)

    // Cancel account ///////////////////////////////////////////////////////////
    // Account cannot be canceled directly
    await expect(accountContract.cancelAccount(consumer1)).to.be.revertedWithCustomError(
      accountContract,
      'MustBePaymentSolution'
    )

    // Account has to be canceled through payment solution (e.g. Prepayment)
    await prepaymentContractConsumerSigner.cancelAccount(id, consumer1)

    // Account was canceled, we cannot access it through account ID anymore
    await expect(prepaymentContractConsumerSigner.getAccount(id)).to.be.revertedWithCustomError(
      prepaymentContractConsumerSigner,
      'InvalidAccount'
    )
  })

  it('Transfer account ownership', async function () {
    const {
      prepaymentContractConsumerSigner,
      consumer: fromConsumer,
      consumer1: toConsumer
    } = await loadFixture(deployPrepayment)
    const txReceipt = await (await prepaymentContractConsumerSigner.createAccount()).wait()

    const accountCreatedEvent = prepaymentContractConsumerSigner.interface.parseLog(
      txReceipt.events[0]
    )
    const { account } = accountCreatedEvent.args
    const accountContract = await ethers.getContractAt('Account', account, fromConsumer)

    // 1. Request Account Transfer
    const requestTxReceipt = await (await accountContract.requestAccountTransfer(toConsumer)).wait()
    expect(requestTxReceipt.events.length).to.be.equal(1)

    const accountTransferRequestedEvent = accountContract.interface.parseLog(
      requestTxReceipt.events[0]
    )
    expect(accountTransferRequestedEvent.name).to.be.equal('AccountTransferRequested')
    const { from: fromRequested, to: toRequested } = accountTransferRequestedEvent.args
    expect(fromRequested).to.be.equal(fromConsumer)
    expect(toRequested).to.be.equal(toConsumer)

    expect(await accountContract.getOwner()).to.be.equal(fromConsumer)
    expect(await accountContract.getRequestedOwner()).to.be.equal(toConsumer)

    // 2.1 Cannot accept with wrong consumer
    await expect(accountContract.acceptAccountTransfer()).to.be.revertedWithCustomError(
      accountContract,
      'MustBeRequestedOwner'
    )

    // 2. Accept Account Transfer
    const accountContractToConsumer = await ethers.getContractAt('Account', account, toConsumer)
    const acceptTxReceipt = await (await accountContractToConsumer.acceptAccountTransfer()).wait()
    expect(acceptTxReceipt.events.length).to.be.equal(1)
    const accountTransferredEvent = accountContractToConsumer.interface.parseLog(
      acceptTxReceipt.events[0]
    )
    expect(accountTransferredEvent.name).to.be.equal('AccountTransferred')

    const { from: fromTransferred, to: toTransferred } = accountTransferredEvent.args
    expect(fromTransferred).to.be.equal(fromConsumer)
    expect(toTransferred).to.be.equal(toConsumer)

    expect(await accountContract.getOwner()).to.be.equal(toConsumer)
    expect(await accountContract.getRequestedOwner()).to.be.equal(
      '0x0000000000000000000000000000000000000000'
    )
  })
})
