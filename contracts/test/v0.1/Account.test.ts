const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')

describe('Account', function () {
  async function deployPrepayment() {
    const {
      deployer,
      consumer,
      consumer1,
      account8: protocolFeeRecipient
    } = await hre.getNamedAccounts()

    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy(protocolFeeRecipient)
    await prepaymentContract.deployed()

    const prepaymentContractConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      consumer
    )

    return { deployer, consumer, consumer1, prepaymentContract, prepaymentContractConsumerSigner }
  }

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
})
