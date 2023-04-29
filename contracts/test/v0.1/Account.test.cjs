const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { cancelAccount, deploy: deployPrepayment } = require('./Prepayment.utils.cjs')

async function createSigners() {
  let { deployer, consumer, consumer1, account8 } = await hre.getNamedAccounts()

  const deployerSigner = await ethers.getSigner(deployer)
  const consumerSigner = await ethers.getSigner(consumer)
  const consumer1Signer = await ethers.getSigner(consumer1)
  const account8Signer = await ethers.getSigner(account8)

  return {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    account8Signer
  }
}

async function deploy() {
  const {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    account8Signer: protocolFeeRecipient
  } = await createSigners()

  const prepaymentContract = await deployPrepayment(protocolFeeRecipient.address, deployerSigner)
  return { deployerSigner, consumerSigner, consumer1Signer, prepaymentContract }
}

describe('Account', function () {
  it('Create & cancel account', async function () {
    const { prepaymentContract, consumerSigner, consumer1Signer } = await loadFixture(deploy)

    // Create account ///////////////////////////////////////////////////////////
    const txReceipt = await (
      await prepaymentContract.connect(consumerSigner).createAccount()
    ).wait()

    expect(txReceipt.events.length).to.be.equal(1)

    const accountCreatedEvent = prepaymentContract
      .connect(consumerSigner)
      .interface.parseLog(txReceipt.events[0])
    expect(accountCreatedEvent.name).to.be.equal('AccountCreated')
    const { accId, account, owner } = accountCreatedEvent.args

    expect(owner).to.be.equal(consumerSigner.address)

    // Access account metadata directly through deployed contract
    const accountContract = await ethers.getContractAt('Account', account, consumerSigner.address)
    const accountOwner = await accountContract.getOwner()
    expect(owner).to.be.equal(accountOwner)

    const accountId = await accountContract.getAccountId()
    expect(accId).to.be.equal(accountId)

    const balance = await accountContract.getBalance()
    expect(balance).to.be.equal(0)

    // Cancel account ///////////////////////////////////////////////////////////
    // Account cannot be canceled directly
    await expect(
      accountContract.cancelAccount(consumer1Signer.address)
    ).to.be.revertedWithCustomError(accountContract, 'MustBePaymentSolution')

    // Account has to be canceled through payment solution (e.g. Prepayment)
    await cancelAccount(prepaymentContract, consumerSigner, accId, consumer1Signer.address)

    // Account was canceled, we cannot access it through account ID anymore
    await expect(
      prepaymentContract.connect(consumerSigner).getAccount(accId)
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidAccount')
  })
})
