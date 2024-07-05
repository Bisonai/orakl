const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { cancelAccount, deploy: deployPrepayment } = require('./Prepayment.utils.cjs')
const { parseAccountCreatedTx, AccountType } = require('./Account.utils.cjs')
const { createSigners, parseKlay } = require('../utils.cjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2: protocolFeeRecipientSigner,
    account3: accountBalanceRecipientSigner,
  } = await createSigners()

  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner,
  )
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner,
  }

  return { consumerSigner, accountBalanceRecipientSigner, prepayment }
}

describe('Account', function () {
  it('Create & cancel account', async function () {
    const { prepayment, consumerSigner, accountBalanceRecipientSigner } = await loadFixture(deploy)

    // Create account ///////////////////////////////////////////////////////////
    const tx = await (await prepayment.contract.connect(consumerSigner).createAccount()).wait()
    const { accId, account, owner, accType } = parseAccountCreatedTx(prepayment, tx)
    expect(owner).to.be.equal(consumerSigner.address)
    expect(accType).to.be.equal(AccountType.KLAY_REGULAR)

    // Access account metadata directly through deployed contract
    const accountContract = await ethers.getContractAt('Account', account, consumerSigner.address)
    expect(await accountContract.getOwner()).to.be.equal(owner)
    expect(await accountContract.getAccountId()).to.be.equal(accId)
    expect(await accountContract.getBalance()).to.be.equal(0)
    expect(await accountContract.typeAndVersion()).to.be.equal('Account v0.1')
    expect(await accountContract.getPaymentSolution()).to.be.equal(prepayment.contract.address)

    //Account can't set fee ratio, update account detail

    await expect(accountContract.setFeeRatio(50)).to.be.revertedWithCustomError(
      accountContract,
      'MustBePaymentSolution',
    )
    const startTime = Math.round(new Date().getTime() / 1000) - 60 * 60
    const period = 60 * 60 * 24 * 7
    const requestNumber = 100
    const subscriptionPrice = parseKlay(100)

    await expect(
      accountContract.updateAccountDetail(startTime, period, requestNumber, subscriptionPrice),
    ).to.be.revertedWithCustomError(accountContract, 'MustBePaymentSolution')

    // Cancel account ///////////////////////////////////////////////////////////
    // Account cannot be canceled directly
    await expect(
      accountContract.cancelAccount(accountBalanceRecipientSigner.address),
    ).to.be.revertedWithCustomError(accountContract, 'MustBePaymentSolution')

    // Account has to be canceled through payment solution (e.g. Prepayment)
    await cancelAccount(
      prepayment.contract,
      consumerSigner,
      accId,
      accountBalanceRecipientSigner.address,
    )

    // Account was canceled, we cannot access it through account ID anymore
    await expect(
      prepayment.contract.connect(consumerSigner).getAccount(accId),
    ).to.be.revertedWithCustomError(prepayment.contract, 'InvalidAccount')
  })
})
