const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { cancelAccount, deploy: deployPrepayment } = require('./Prepayment.utils.cjs')
const { parseAccountCreatedTx } = require('./Account.utils.cjs')
const { createSigners } = require('./utils.cjs')

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2: protocolFeeRecipientSigner,
    account3: accountBalanceRecipientSigner
  } = await createSigners()

  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner
  )
  const prepayment = {
    contract: prepaymentContract,
    signer: deployerSigner
  }

  return { consumerSigner, accountBalanceRecipientSigner, prepayment }
}

describe('Account', function () {
  it('Create & cancel account', async function () {
    const { prepayment, consumerSigner, accountBalanceRecipientSigner } = await loadFixture(deploy)

    // Create account ///////////////////////////////////////////////////////////
    const tx = await (await prepayment.contract.connect(consumerSigner).createAccount()).wait()
    const { accId, account, owner } = parseAccountCreatedTx(prepayment, tx)
    expect(owner).to.be.equal(consumerSigner.address)

    // Access account metadata directly through deployed contract
    const accountContract = await ethers.getContractAt('Account', account, consumerSigner.address)
    expect(await accountContract.getOwner()).to.be.equal(owner)
    expect(await accountContract.getAccountId()).to.be.equal(accId)
    expect(await accountContract.getBalance()).to.be.equal(0)
    expect(await accountContract.typeAndVersion()).to.be.equal('Account v0.1')
    expect(await accountContract.getPaymentSolution()).to.be.equal(prepayment.contract.address)

    // Cancel account ///////////////////////////////////////////////////////////
    // Account cannot be canceled directly
    await expect(
      accountContract.cancelAccount(accountBalanceRecipientSigner.address)
    ).to.be.revertedWithCustomError(accountContract, 'MustBePaymentSolution')

    // Account has to be canceled through payment solution (e.g. Prepayment)
    await cancelAccount(
      prepayment.contract,
      consumerSigner,
      accId,
      accountBalanceRecipientSigner.address
    )

    // Account was canceled, we cannot access it through account ID anymore
    await expect(
      prepayment.contract.connect(consumerSigner).getAccount(accId)
    ).to.be.revertedWithCustomError(prepayment.contract, 'InvalidAccount')
  })
})
