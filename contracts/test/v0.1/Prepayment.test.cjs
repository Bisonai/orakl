const { expect } = require('chai')
const { ethers } = require('hardhat')
const { loadFixture } = require('@nomicfoundation/hardhat-network-helpers')
const { deploy: deployPrepayment, createAccount, deposit } = require('./Prepayment.utils.cjs')
const { parseKlay } = require('./utils.cjs')

const NULL_ADDRESS = '0x0000000000000000000000000000000000000000'
const DEFAULT_BURN_FEE_RATIO = 50
const DEFAULT_PROTOCOL_FEE_RATIO = 5

async function createSigners() {
  let { deployer, consumer, consumer1, consumer2, account8 } = await hre.getNamedAccounts()

  const deployerSigner = await ethers.getSigner(deployer)
  const consumerSigner = await ethers.getSigner(consumer)
  const consumer1Signer = await ethers.getSigner(consumer1)
  const consumer2Signer = await ethers.getSigner(consumer2)
  const account8Signer = await ethers.getSigner(account8)

  return {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    consumer2Signer,
    account8Signer
  }
}

async function deploy() {
  const {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    consumer2Signer,
    account8Signer: protocolFeeRecipientSigner
  } = await createSigners()

  const prepaymentContract = await deployPrepayment(
    protocolFeeRecipientSigner.address,
    deployerSigner
  )

  return {
    deployerSigner,
    consumerSigner,
    consumer1Signer,
    consumer2Signer,
    prepaymentContract,
    protocolFeeRecipientSigner
  }
}

describe('Prepayment', function () {
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
      prepaymentContract.setBurnFeeRatio(higherThresholdRatio)
    ).to.be.revertedWithCustomError(prepaymentContract, 'TooHighFeeRatio')

    // 3. Set burnFee ratio with
    const ratioBelowThreshold = -1
    await expect(prepaymentContract.setBurnFeeRatio(ratioBelowThreshold)).to.be.rejected

    const ratioAboveThreshold = 101
    await expect(
      prepaymentContract.setBurnFeeRatio(ratioAboveThreshold)
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
      prepaymentContract.setProtocolFeeRatio(higherThresholdRatio)
    ).to.be.revertedWithCustomError(prepaymentContract, 'TooHighFeeRatio')

    // 3. Set burn ratio with
    const ratioBelowThreshold = -1
    await expect(prepaymentContract.setProtocolFeeRatio(ratioBelowThreshold)).to.be.rejected

    const ratioAboveThreshold = 101
    await expect(
      prepaymentContract.setProtocolFeeRatio(ratioAboveThreshold)
    ).to.be.revertedWithCustomError(prepaymentContract, 'RatioOutOfBounds')
  })

  it('Protocol fee recipient setup', async function () {
    const {
      prepaymentContract,
      protocolFeeRecipientSigner,
      consumer2Signer: newProtocolFeeRecipientSigner
    } = await loadFixture(deploy)
    const recipient = await prepaymentContract.getProtocolFeeRecipient()
    expect(recipient).to.be.equal(protocolFeeRecipientSigner.address)

    // onlyOwner can set protocol fee recipient
    await expect(
      prepaymentContract
        .connect(newProtocolFeeRecipientSigner)
        .setProtocolFeeRecipient(newProtocolFeeRecipientSigner.address)
    ).to.be.rejected

    // update protocol fee recipient
    await prepaymentContract.setProtocolFeeRecipient(newProtocolFeeRecipientSigner.address)
    expect(await prepaymentContract.getProtocolFeeRecipient()).to.be.equal(
      newProtocolFeeRecipientSigner.address
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
      consumer1Signer: nonOwner
    } = await loadFixture(deploy)

    const { accId, account } = await createAccount(prepaymentContract, accountOwner)
    const accountContract = await ethers.getContractAt('Account', account, accountOwner.address)

    // Get Balance
    const balanceBefore = (await accountContract.getBalance()).toNumber()
    expect(balanceBefore).to.be.equal(0)

    // 1. Deposit $KLAY /////////////////////////////////////////////////////////
    const amount = parseKlay(10)
    const { oldBalance, newBalance } = await deposit(
      prepaymentContract,
      accountOwner,
      accId,
      amount
    )
    const balanceAfterDeposit = await accountContract.getBalance()
    expect(balanceBefore).to.be.equal(oldBalance)
    expect(balanceAfterDeposit).to.be.equal(newBalance)

    // 2. Withdraw $KLAY ////////////////////////////////////////////////////////
    // Only account owner can withdraw
    await expect(
      prepaymentContract.connect(nonOwner).withdraw(accId, amount)
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Withdrawing using the account owner
    const txWithdraw = await (
      await prepaymentContract.connect(accountOwner).withdraw(accId, amount)
    ).wait()

    // All previously deposited $KLAY were withdrawn. Nothing is left.
    const balanceAfterWithdraw = (await accountContract.getBalance()).toNumber()
    expect(balanceAfterWithdraw).to.be.equal(0)

    // Check the event information
    expect(txWithdraw.events.length).to.be.equal(1)
    const accountBalanceDecreasedEvent = prepaymentContract.interface.parseLog(txWithdraw.events[0])
    expect(accountBalanceDecreasedEvent.name).to.be.equal('AccountBalanceDecreased')
    const {
      accId: accIdWithdraw,
      oldBalance: oldBalanceWithdraw,
      newBalance: newBalanceWithdraw
    } = accountBalanceDecreasedEvent.args
    expect(accIdWithdraw).to.be.equal(accId)
    expect(balanceAfterDeposit).to.be.equal(oldBalanceWithdraw)
    expect(balanceBefore).to.be.equal(newBalanceWithdraw)
  })

  it('Add & remove consumer', async function () {
    const {
      prepaymentContract,
      consumerSigner: accountOwner,
      consumer1Signer: consumer,
      consumer1Signer: nonOwner,
      consumer2Signer: unusedConsumer
    } = await loadFixture(deploy)

    const { accId, account } = await createAccount(prepaymentContract, accountOwner)

    const accountContract = await ethers.getContractAt('Account', account, accountOwner.address)
    expect((await accountContract.getConsumers()).length).to.be.equal(0)

    // 1. Add consumer //////////////////////////////////////////////////////////
    // Consumer can be added only by the account owner
    await expect(
      prepaymentContract.connect(nonOwner).addConsumer(accId, consumer.address)
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Add consumer with correct signer and parameters
    const txReceiptAddConsumer = await (
      await prepaymentContract.connect(accountOwner).addConsumer(accId, consumer.address)
    ).wait()
    expect(txReceiptAddConsumer.events.length).to.be.equal(1)

    const accountConsumerAddedEvent = prepaymentContract.interface.parseLog(
      txReceiptAddConsumer.events[0]
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
      prepaymentContract.connect(accountOwner).removeConsumer(accId, unusedConsumer.address)
    ).to.be.revertedWithCustomError(accountContract, 'InvalidConsumer')

    // Consumer can be removed only by the account owner
    await expect(
      prepaymentContract.connect(nonOwner).removeConsumer(accId, consumer.address)
    ).to.be.revertedWithCustomError(prepaymentContract, 'MustBeAccountOwner')

    // Remove consumer with correct signer and paramters
    const txReceiptRemoveConsumer = await (
      await prepaymentContract.connect(accountOwner).removeConsumer(accId, consumer.address)
    ).wait()
    expect(txReceiptRemoveConsumer.events.length).to.be.equal(1)

    const accountConsumerRemovedEvent = prepaymentContract.interface.parseLog(
      txReceiptRemoveConsumer.events[0]
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
    const {
      consumerSigner,
      prepaymentContract,
      consumer1Signer: coordinator
    } = await loadFixture(deploy)

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
    await expect(prepaymentContract.addCoordinator(coordinator.address)).to.be.rejectedWith(
      'CoordinatorExists'
    )

    // Remove coordinator ///////////////////////////////////////////////////////
    // Non-existing coordinator cannot be removed
    await expect(prepaymentContract.removeCoordinator(NULL_ADDRESS)).to.be.rejectedWith(
      'InvalidCoordinator'
    )

    // We can remove coordinator that has been previously added
    const txRemoveCoordinator = await (
      await prepaymentContract.removeCoordinator(coordinator.address)
    ).wait()
    expect((await prepaymentContract.getCoordinators()).length).to.be.equal(0)

    // Check the event information
    expect(txRemoveCoordinator.events.length).to.be.equal(1)
    const removeCoordinatorEvent = prepaymentContract.interface.parseLog(
      txRemoveCoordinator.events[0]
    )
    expect(removeCoordinatorEvent.name).to.be.equal('CoordinatorRemoved')
    const { coordinator: removeCoordinatorAddress } = removeCoordinatorEvent.args
    expect(removeCoordinatorAddress).to.be.equal(coordinator.address)
  })

  it('Transfer account ownership', async function () {
    const {
      consumerSigner: fromConsumer,
      consumer1Signer: toConsumer,
      prepaymentContract
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
      requestTxReceipt.events[0]
    )
    expect(accountTransferRequestedEvent.name).to.be.equal('AccountOwnerTransferRequested')
    const { from: fromRequested, to: toRequested } = accountTransferRequestedEvent.args
    expect(fromRequested).to.be.equal(fromConsumer.address)
    expect(toRequested).to.be.equal(toConsumer.address)

    expect(await accountContract.getOwner()).to.be.equal(fromConsumer.address)
    expect(await accountContract.getRequestedOwner()).to.be.equal(toConsumer.address)

    // 2.1 Cannot accept with wrong consumer
    await expect(
      prepaymentContract.connect(fromConsumer).acceptAccountOwnerTransfer(accId)
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
    const {
      deployerSigner,
      consumerSigner,
      account8Signer: protocolFeeRecipientSigner
    } = await createSigners()

    const prepaymentContract = await deployPrepayment(
      protocolFeeRecipientSigner.address,
      deployerSigner
    )

    const { accId } = await createAccount(prepaymentContract, consumerSigner)
    const initialAmount = parseKlay(1)
    await deposit(prepaymentContract, consumerSigner, accId, initialAmount)

    const aboveBalance = initialAmount + parseKlay(1)
    const accountContract = await ethers.getContractFactory('Account')
    await expect(
      prepaymentContract.connect(consumerSigner).withdraw(accId, aboveBalance)
    ).to.be.revertedWithCustomError(accountContract, 'InsufficientBalance')
  })

  it('Withdraw pending request exists', async function () {})
})
