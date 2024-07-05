const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const {
  deploy: deployPrepayment,
  createAccount,
  deposit,
  withdraw,
} = require('./Prepayment.utils.cjs')
const { parseKlay, getBalance, createSigners } = require('../utils.cjs')

const NULL_ADDRESS = '0x0000000000000000000000000000000000000000'
const DEFAULT_BURN_FEE_RATIO = 50
const DEFAULT_PROTOCOL_FEE_RATIO = 5
const DEFAULT_ACCOUNT_FEE_RATIO = 0

async function deploy() {
  const {
    account0: deployerSigner,
    account1: consumerSigner,
    account2: protocolFeeRecipientSigner,
    account3,
    account4,
    account5,
  } = await createSigners()

  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner,
  )

  return {
    deployerSigner,
    consumerSigner,
    protocolFeeRecipientSigner,
    account3,
    account4,
    account5,
    prepaymentContract,
  }
}

describe('Prepayment', function () {
  it('Type and version', async function () {
    const { prepaymentContract } = await loadFixture(deploy)
    expect(await prepaymentContract.typeAndVersion()).to.be.equal('Prepayment v0.1')
  })

  it('Burn ratio setup', async function () {
    const { prepaymentContract } = await loadFixture(deploy)

    // 1. Get initial burn ratio
    const burnFeeRatio = await prepaymentContract.getBurnFeeRatio()
    expect(burnFeeRatio).to.be.equal(DEFAULT_BURN_FEE_RATIO)

    // 2. Set burnFee ratio
    const lowerThresholdRatio = 0
    await prepaymentContract.setBurnFeeRatio(lowerThresholdRatio)
    expect(await prepaymentContract.getBurnFeeRatio()).to.be.equal(lowerThresholdRatio)

    const higherThresholdRatio = 100
    await expect(
      prepaymentContract.setBurnFeeRatio(higherThresholdRatio),
    ).to.be.revertedWithCustomError(prepaymentContract, 'TooHighFeeRatio')

    // 3. Set burnFee ratio with
    const ratioBelowThreshold = -1
    await expect(prepaymentContract.setBurnFeeRatio(ratioBelowThreshold)).to.be.rejected

    const ratioAboveThreshold = 101
    await expect(
      prepaymentContract.setBurnFeeRatio(ratioAboveThreshold),
    ).to.be.revertedWithCustomError(prepaymentContract, 'RatioOutOfBounds')
  })

  it('Protocol fee ratio setup', async function () {
    const { prepaymentContract } = await loadFixture(deploy)

    // 1. Get initial burn ratio
    const protocolFeeRatio = await prepaymentContract.getProtocolFeeRatio()
    expect(protocolFeeRatio).to.be.equal(DEFAULT_PROTOCOL_FEE_RATIO)

    // 2. Set protocolFee ratio
    const lowerThresholdRatio = 0
    await prepaymentContract.setProtocolFeeRatio(lowerThresholdRatio)
    expect(await prepaymentContract.getProtocolFeeRatio()).to.be.equal(lowerThresholdRatio)

    const higherThresholdRatio = 100
    await expect(
      prepaymentContract.setProtocolFeeRatio(higherThresholdRatio),
    ).to.be.revertedWithCustomError(prepaymentContract, 'TooHighFeeRatio')

    // 3. Set burn ratio with
    const ratioBelowThreshold = -1
    await expect(prepaymentContract.setProtocolFeeRatio(ratioBelowThreshold)).to.be.rejected

    const ratioAboveThreshold = 101
    await expect(
      prepaymentContract.setProtocolFeeRatio(ratioAboveThreshold),
    ).to.be.revertedWithCustomError(prepaymentContract, 'RatioOutOfBounds')
  })

  it('Account fee ratio setup', async function () {
    const { prepaymentContract, consumerSigner: accountOwner } = await loadFixture(deploy)
    const { accId } = await createAccount(prepaymentContract, accountOwner)
    // 1. Get initial fee ratio
    const accountFeeRatio = await prepaymentContract.getFeeRatio(accId)
    expect(accountFeeRatio).to.be.equal(DEFAULT_ACCOUNT_FEE_RATIO)

    // 2. Set fee ratio
    const lowerThresholdRatio = 0
    await prepaymentContract.setFeeRatio(accId, lowerThresholdRatio)
    expect(await prepaymentContract.getFeeRatio(accId)).to.be.equal(lowerThresholdRatio)

    // 3. Set fee ratio with
    const ratioBelowThreshold = -1
    await expect(prepaymentContract.setFeeRatio(accId, ratioBelowThreshold)).to.be.rejected
  })

  it('Protocol fee recipient setup', async function () {
    const {
      prepaymentContract,
      protocolFeeRecipientSigner,
      account4: newProtocolFeeRecipientSigner,
    } = await loadFixture(deploy)
    const recipient = await prepaymentContract.getProtocolFeeRecipient()
    expect(recipient).to.be.equal(protocolFeeRecipientSigner.address)

    // onlyOwner can set protocol fee recipient
    await expect(
      prepaymentContract
        .connect(newProtocolFeeRecipientSigner)
        .setProtocolFeeRecipient(newProtocolFeeRecipientSigner.address),
    ).to.be.rejected

    // update protocol fee recipient
    await prepaymentContract.setProtocolFeeRecipient(newProtocolFeeRecipientSigner.address)
    expect(await prepaymentContract.getProtocolFeeRecipient()).to.be.equal(
      newProtocolFeeRecipientSigner.address,
    )
  })

  it('Account owner', async function () {
    const { prepaymentContract, consumerSigner: accountOwnerSigner } = await loadFixture(deploy)
    const { accId, account } = await createAccount(prepaymentContract, accountOwnerSigner)
    const accountOwner = await prepaymentContract.getAccountOwner(accId)
    expect(accountOwner).to.be.equal(accountOwnerSigner.address)
  })

  it('Deposit & withdraw', async function () {
    const {
      prepaymentContract,
      consumerSigner: accountOwner,
      account3: nonOwner,
    } = await loadFixture(deploy)

    const { accId, account } = await createAccount(prepaymentContract, accountOwner)
    const accountContract = await ethers.getContractAt('Account', account, accountOwner.address)

    // Get Balance
    const balanceBeforeDeposit = (await accountContract.getBalance()).toNumber()
    expect(balanceBeforeDeposit).to.be.equal(0)

    // 1. Deposit $KLAY /////////////////////////////////////////////////////////
    const amount = parseKlay(10)
    const { oldBalance: oldBalanceDeposit, newBalance: newBalanceDeposit } = await deposit(
      prepaymentContract,
      accountOwner,
      accId,
      amount,
    )

    // Read balance directly from Account contract
    const balanceAfterDeposit = await accountContract.getBalance()
    expect(balanceBeforeDeposit).to.be.equal(oldBalanceDeposit)
    expect(balanceAfterDeposit).to.be.equal(newBalanceDeposit)

    // Read balance indirectly through Prepayment contract
    const prepaymentBalanceAfterDeposit = await prepaymentContract.getBalance(accId)
    expect(prepaymentBalanceAfterDeposit).to.be.equal(balanceAfterDeposit)

    // 2. Withdraw $KLAY ////////////////////////////////////////////////////////
    // Only account owner can withdraw
    await expect(
      prepaymentContract.connect(nonOwner).withdraw(accId, amount),
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Withdrawing using the account owner
    const { oldBalance: oldBalanceWithdraw, newBalance: newBalanceWithdraw } = await withdraw(
      prepaymentContract,
      accountOwner,
      accId,
      amount,
    )
    expect(balanceAfterDeposit).to.be.equal(oldBalanceWithdraw)
    expect(balanceBeforeDeposit).to.be.equal(newBalanceWithdraw)

    // All previously deposited $KLAY were withdrawn. Nothing is left.
    const balanceAfterWithdraw = (await accountContract.getBalance()).toNumber()
    expect(balanceAfterWithdraw).to.be.equal(0)
  })

  it('Deposit to non-existant account', async function () {
    // It is not possible to deposit to non-existant account, deposit
    // transaction reverts in such case.
    const { prepaymentContract, consumerSigner: accountOwner } = await loadFixture(deploy)
    const accId = 123
    const amount = parseKlay(10)
    await expect(
      deposit(prepaymentContract, accountOwner, accId, amount),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidAccount')
  })

  it('Cannot deposit more than current balance', async function () {
    const { prepaymentContract, consumerSigner: accountOwner } = await loadFixture(deploy)
    const { accId } = await createAccount(prepaymentContract, accountOwner)
    const amount = (await getBalance(accountOwner.address)) + parseKlay(1)
    /* const amount = parseKlay(10_001) */
    await expect(
      prepaymentContract.connect(accountOwner).deposit(accId, {
        value: amount,
      }),
    ).to.be.rejected
  })

  it('Add & remove consumer', async function () {
    const {
      prepaymentContract,
      consumerSigner: accountOwner,
      account3: consumer,
      account4: nonOwner,
      account5: unusedConsumer,
    } = await loadFixture(deploy)

    const { accId, account } = await createAccount(prepaymentContract, accountOwner)

    const accountContract = await ethers.getContractAt('Account', account, accountOwner.address)
    expect((await accountContract.getConsumers()).length).to.be.equal(0)

    // 1. Add consumer //////////////////////////////////////////////////////////
    // Consumer can be added only by the account owner
    await expect(
      prepaymentContract.connect(nonOwner).addConsumer(accId, consumer.address),
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Add consumer with correct signer and parameters
    const txReceiptAddConsumer = await (
      await prepaymentContract.connect(accountOwner).addConsumer(accId, consumer.address)
    ).wait()
    expect(txReceiptAddConsumer.events.length).to.be.equal(1)

    const accountConsumerAddedEvent = prepaymentContract.interface.parseLog(
      txReceiptAddConsumer.events[0],
    )
    expect(accountConsumerAddedEvent.name).to.be.equal('AccountConsumerAdded')
    const { accId: consumerAddedAccId, consumer: consumerAddedConsumer } =
      accountConsumerAddedEvent.args
    expect(consumerAddedAccId).to.be.equal(accId)
    expect(consumerAddedConsumer).to.be.equal(consumer.address)

    // Idempotance - adding the same consumer does not do anything
    const consumersBefore = (await accountContract.getConsumers()).length
    await prepaymentContract.connect(accountOwner).addConsumer(accId, consumer.address)
    const consumersAfter = (await accountContract.getConsumers()).length
    expect(consumersBefore).to.be.equal(consumersAfter)

    // Consumers can be access directly through `Account.getConsumers`.
    // Now, there should be single consumer.
    expect((await accountContract.getConsumers()).length).to.be.equal(1)

    // 2. Remove consumer ///////////////////////////////////////////////////////
    // We cannot remove consumer which is not there
    await expect(
      prepaymentContract.connect(accountOwner).removeConsumer(accId, unusedConsumer.address),
    ).to.be.revertedWithCustomError(accountContract, 'InvalidConsumer')

    // Consumer can be removed only by the account owner
    await expect(
      prepaymentContract.connect(nonOwner).removeConsumer(accId, consumer.address),
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Remove consumer with correct signer and paramters
    const txReceiptRemoveConsumer = await (
      await prepaymentContract.connect(accountOwner).removeConsumer(accId, consumer.address)
    ).wait()
    expect(txReceiptRemoveConsumer.events.length).to.be.equal(1)

    const accountConsumerRemovedEvent = prepaymentContract.interface.parseLog(
      txReceiptRemoveConsumer.events[0],
    )
    expect(accountConsumerRemovedEvent.name).to.be.equal('AccountConsumerRemoved')
    const { accId: consumerRemovedAccId, consumer: consumerRemovedConsumer } =
      accountConsumerRemovedEvent.args

    expect(consumerRemovedAccId).to.be.equal(accId)
    expect(consumerRemovedConsumer).to.be.equal(consumer.address)

    // After removing the consumer, there should no consumer anymore
    expect((await accountContract.getConsumers()).length).to.be.equal(0)
  })

  it('Add & remove coordinator', async function () {
    const { consumerSigner, prepaymentContract, account3: coordinator } = await loadFixture(deploy)

    // Add coordinator //////////////////////////////////////////////////////////
    // Coordinator must be added by contract owner
    await expect(prepaymentContract.connect(consumerSigner).addCoordinator(coordinator.address)).to
      .be.rejected

    const txAddCoordinator = await (
      await prepaymentContract.addCoordinator(coordinator.address)
    ).wait()

    // Check the event information
    expect(txAddCoordinator.events.length).to.be.equal(1)
    const addCoordinatorEvent = prepaymentContract.interface.parseLog(txAddCoordinator.events[0])
    expect(addCoordinatorEvent.name).to.be.equal('CoordinatorAdded')
    const { coordinator: addedCoordinatorAddress } = addCoordinatorEvent.args
    expect(addedCoordinatorAddress).to.be.equal(coordinator.address)

    // The same coordinator cannot be added more than once
    await expect(
      prepaymentContract.addCoordinator(coordinator.address),
    ).to.be.revertedWithCustomError(prepaymentContract, 'CoordinatorExists')

    // Remove coordinator ///////////////////////////////////////////////////////
    // Non-existing coordinator cannot be removed
    await expect(prepaymentContract.removeCoordinator(NULL_ADDRESS)).to.be.revertedWithCustomError(
      prepaymentContract,
      'InvalidCoordinator',
    )

    // We can remove coordinator that has been previously added
    const txRemoveCoordinator = await (
      await prepaymentContract.removeCoordinator(coordinator.address)
    ).wait()
    expect((await prepaymentContract.getCoordinators()).length).to.be.equal(0)

    // Check the event information
    expect(txRemoveCoordinator.events.length).to.be.equal(1)
    const removeCoordinatorEvent = prepaymentContract.interface.parseLog(
      txRemoveCoordinator.events[0],
    )
    expect(removeCoordinatorEvent.name).to.be.equal('CoordinatorRemoved')
    const { coordinator: removeCoordinatorAddress } = removeCoordinatorEvent.args
    expect(removeCoordinatorAddress).to.be.equal(coordinator.address)
  })

  it('Transfer account ownership', async function () {
    const {
      consumerSigner: fromConsumer,
      account3: toConsumer,
      prepaymentContract,
    } = await loadFixture(deploy)

    const { accId, account } = await createAccount(prepaymentContract, fromConsumer)
    const accountContract = await ethers.getContractAt('Account', account, fromConsumer.address)

    // 1. Request Account Transfer
    const requestTxReceipt = await (
      await prepaymentContract
        .connect(fromConsumer)
        .requestAccountOwnerTransfer(accId, toConsumer.address)
    ).wait()
    expect(requestTxReceipt.events.length).to.be.equal(1)

    const accountTransferRequestedEvent = prepaymentContract.interface.parseLog(
      requestTxReceipt.events[0],
    )
    expect(accountTransferRequestedEvent.name).to.be.equal('AccountOwnerTransferRequested')
    const { from: fromRequested, to: toRequested } = accountTransferRequestedEvent.args
    expect(fromRequested).to.be.equal(fromConsumer.address)
    expect(toRequested).to.be.equal(toConsumer.address)

    expect(await accountContract.getOwner()).to.be.equal(fromConsumer.address)
    expect(await accountContract.getRequestedOwner()).to.be.equal(toConsumer.address)

    // 2.1 Cannot accept with wrong consumer
    await expect(
      prepaymentContract.connect(fromConsumer).acceptAccountOwnerTransfer(accId),
    ).to.be.revertedWithCustomError(accountContract, 'MustBeRequestedOwner')

    // 2. Accept Account Transfer
    const acceptTxReceipt = await (
      await prepaymentContract.connect(toConsumer).acceptAccountOwnerTransfer(accId)
    ).wait()
    expect(acceptTxReceipt.events.length).to.be.equal(1)
    const accountTransferredEvent = prepaymentContract.interface.parseLog(acceptTxReceipt.events[0])
    expect(accountTransferredEvent.name).to.be.equal('AccountOwnerTransferred')

    const { from: fromTransferred, to: toTransferred } = accountTransferredEvent.args
    expect(fromTransferred).to.be.equal(fromConsumer.address)
    expect(toTransferred).to.be.equal(toConsumer.address)

    expect(await accountContract.getOwner()).to.be.equal(toConsumer.address)
    expect(await accountContract.getRequestedOwner()).to.be.equal(NULL_ADDRESS)
  })

  it('Try to withdraw more than current balance', async function () {
    const { deployerSigner, consumerSigner, protocolFeeRecipientSigner } = await loadFixture(deploy)

    const prepaymentContract = await deployPrepayment(
      protocolFeeRecipientSigner.address,
      deployerSigner,
    )

    const { accId } = await createAccount(prepaymentContract, consumerSigner)
    const initialAmount = parseKlay(1)
    await deposit(prepaymentContract, consumerSigner, accId, initialAmount)

    const aboveBalance = initialAmount + parseKlay(1)
    const accountContract = await ethers.getContractFactory('Account')
    await expect(
      prepaymentContract.connect(consumerSigner).withdraw(accId, aboveBalance),
    ).to.be.revertedWithCustomError(accountContract, 'InsufficientBalance')
  })

  it('TooManyConsumers', async function () {
    const { deployerSigner, consumerSigner, protocolFeeRecipientSigner } = await loadFixture(deploy)

    const prepaymentContract = await deployPrepayment(
      protocolFeeRecipientSigner.address,
      deployerSigner,
    )

    const { accId, account } = await createAccount(prepaymentContract, consumerSigner)
    const accountContract = await ethers.getContractAt('Account', account, consumerSigner.address)
    const MAX_CONSUMERS = await accountContract.MAX_CONSUMERS()

    for (let i = 0; i < MAX_CONSUMERS; ++i) {
      const { address: consumer } = ethers.Wallet.createRandom()
      await prepaymentContract.connect(consumerSigner).addConsumer(accId, consumer)
    }

    // There is a limit (MAX_CONSUMERS) on number of consumers that can be added
    const { address: consumer } = ethers.Wallet.createRandom()
    await expect(
      prepaymentContract.connect(consumerSigner).addConsumer(accId, consumer),
    ).to.be.revertedWithCustomError(accountContract, 'TooManyConsumers')
  })

  it('OnlyOwner', async function () {
    const { prepaymentContract, consumerSigner } = await loadFixture(deploy)

    await expect(prepaymentContract.connect(consumerSigner).setBurnFeeRatio(5)).to.be.revertedWith(
      'Ownable: caller is not the owner',
    )

    await expect(
      prepaymentContract.connect(consumerSigner).setProtocolFeeRatio(5),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    await expect(
      prepaymentContract.connect(consumerSigner).setProtocolFeeRecipient(consumerSigner.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    await expect(
      prepaymentContract.connect(consumerSigner).addCoordinator(consumerSigner.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    await expect(
      prepaymentContract.connect(consumerSigner).removeCoordinator(consumerSigner.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    const { accId } = await createAccount(prepaymentContract, consumerSigner)
    //set fee ratio for account
    await expect(
      prepaymentContract.connect(consumerSigner).setFeeRatio(accId, 50),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    //update account detail
    const startTime = Math.round(new Date().getTime() / 1000) - 60 * 60
    const period = 60 * 60 * 24 * 7
    const requestNumber = 100
    const subscriptionPrice = 0
    const feeRatio = 10000 // 100%

    await expect(
      prepaymentContract
        .connect(consumerSigner)
        .updateAccountDetail(accId, startTime, period, requestNumber, subscriptionPrice),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    // create fiat, klay subscription and discount account
    await expect(
      prepaymentContract
        .connect(consumerSigner)
        .createFiatSubscriptionAccount(startTime, period, requestNumber, consumerSigner.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    await expect(
      prepaymentContract
        .connect(consumerSigner)
        .createKlaySubscriptionAccount(
          startTime,
          period,
          requestNumber,
          subscriptionPrice,
          consumerSigner.address,
        ),
    ).to.be.revertedWith('Ownable: caller is not the owner')

    await expect(
      prepaymentContract
        .connect(consumerSigner)
        .createKlayDiscountAccount(feeRatio, consumerSigner.address),
    ).to.be.revertedWith('Ownable: caller is not the owner')
  })

  it('OnlyCoordinator', async function () {
    const { prepaymentContract, consumerSigner } = await loadFixture(deploy)

    await expect(
      prepaymentContract.connect(consumerSigner).chargeFee(1, 1_000),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')

    await expect(
      prepaymentContract.connect(consumerSigner).chargeFeeTemporary(1),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')

    await expect(
      prepaymentContract
        .connect(consumerSigner)
        .chargeOperatorFee(1, 1_000, consumerSigner.address),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')

    await expect(
      prepaymentContract.connect(consumerSigner).increaseNonce(1, consumerSigner.address),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')

    await expect(
      prepaymentContract.connect(consumerSigner).increaseSubReqCount(1),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')

    await expect(
      prepaymentContract.connect(consumerSigner).setSubscriptionPaid(1),
    ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')
  })
})
