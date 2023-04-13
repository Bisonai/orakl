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

  // it('Get & increase nonce', async function () {
  //   const {
  //     prepaymentContractConsumerSigner,
  //     consumer: accountOwnerAddress,
  //     consumer1: consumerAddress
  //   } = await loadFixture(deployPrepayment)
  //
  //   const txReceipt = await (await prepaymentContractConsumerSigner.createAccount()).wait()
  //   const accountCreatedEvent = prepaymentContractConsumerSigner.interface.parseLog(
  //     txReceipt.events[0]
  //   )
  //   const { account } = accountCreatedEvent.args
  //   const accountContract = await ethers.getContractAt('Account', account, accountOwnerAddress)
  //
  //   // Querying nonce without registered consumer is possible but will always return 0
  //   const nonce = (await accountContract.getNonce(consumerAddress)).toNumber()
  //   expect(nonce).to.be.equal(0)
  // })
})
