import { expect } from 'chai'
import { ethers } from 'hardhat'
import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'

const NULL_ADDRESS = '0x0000000000000000000000000000000000000000'
const DEFAULT_BURN_FEE_RATIO = 50
const DEFAULT_PROTOCOL_FEE_RATIO = 5

describe('Prepayment', function () {
  async function deployPrepayment() {
    const {
      deployer,
      consumer,
      consumer1,
      consumer2,
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

    return {
      deployer,
      consumer,
      consumer1,
      consumer2,
      prepaymentContract,
      prepaymentContractConsumerSigner
    }
  }

  it('Burn ratio setup', async function () {
    const { prepaymentContract } = await loadFixture(deployPrepayment)

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
    const { prepaymentContract } = await loadFixture(deployPrepayment)

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

  it('Deposit & withdraw', async function () {
    const {
      prepaymentContractConsumerSigner: prepaymentContract,
      consumer: accountOwnerAddress,
      consumer1: nonOwnerAddress
    } = await loadFixture(deployPrepayment)

    const txReceipt = await (await prepaymentContract.createAccount()).wait()
    const accountCreatedEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
    const { accId, account } = accountCreatedEvent.args

    const accountContract = await ethers.getContractAt('Account', account, accountOwnerAddress)

    // Get Balance
    const balanceBefore = (await accountContract.getBalance()).toNumber()
    expect(balanceBefore).to.be.equal(0)

    // 1. Deposit $KLAY /////////////////////////////////////////////////////////
    const amount = 10
    const txDeposit = await (await prepaymentContract.deposit(accId, { value: amount })).wait()
    const balanceAfterDeposit = (await accountContract.getBalance()).toNumber()
    expect(balanceAfterDeposit).to.be.equal(amount)

    // Check the event information
    expect(txDeposit.events.length).to.be.equal(1)
    const accountBalanceIncreasedEvent = prepaymentContract.interface.parseLog(txDeposit.events[0])
    expect(accountBalanceIncreasedEvent.name).to.be.equal('AccountBalanceIncreased')
    const { accId: accIdDeposit, oldBalance, newBalance } = accountBalanceIncreasedEvent.args
    expect(accIdDeposit).to.be.equal(accId)
    expect(balanceBefore).to.be.equal(oldBalance)
    expect(balanceAfterDeposit).to.be.equal(newBalance)

    // 2. Withdraw $KLAY ////////////////////////////////////////////////////////
    // Only account owner can withdraw
    const prepaymentContractNonOwnerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      nonOwnerAddress
    )
    await expect(
      prepaymentContractNonOwnerSigner.withdraw(accId, amount)
    ).to.be.revertedWithCustomError(prepaymentContractNonOwnerSigner, 'MustBeAccountOwner')

    // Withdrawing using the account owner
    const txWithdraw = await (await prepaymentContract.withdraw(accId, amount)).wait()

    // All previously deposited $KLAY was withdrawn. Nothin is left.
    const balanceAfterWithdraw = (await accountContract.getBalance()).toNumber()
    expect(balanceAfterWithdraw).to.be.equal(0)

    // Check the event information
    expect(txWithdraw.events.length).to.be.equal(1)
    const accountBalanceDecreasedEvent = prepaymentContract.interface.parseLog(txWithdraw.events[0])
    expect(accountBalanceDecreasedEvent.name).to.be.equal('AccountBalanceDecreased')
    const {
      accId: accIdWithdraw,
      oldBalance: oldBalanceWithdraw,
      newBalance: newBalanceWithdraw,
      burnAmount
    } = accountBalanceDecreasedEvent.args
    expect(accIdWithdraw).to.be.equal(accId)
    expect(balanceAfterDeposit).to.be.equal(oldBalanceWithdraw)
    expect(balanceBefore).to.be.equal(newBalanceWithdraw)
    expect(burnAmount).to.be.equal(0)
  })

  it('Add & remove consumer', async function () {
    const {
      prepaymentContractConsumerSigner: prepaymentContract,
      consumer: accountOwnerAddress,
      consumer1: consumerAddress,
      consumer1: nonOwnerAddress,
      consumer2: unusedConsumer
    } = await loadFixture(deployPrepayment)

    const txReceipt = await (await prepaymentContract.createAccount()).wait()
    const accountCreatedEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
    const { accId, account } = accountCreatedEvent.args
    const accountContract = await ethers.getContractAt('Account', account, accountOwnerAddress)
    expect((await accountContract.getConsumers()).length).to.be.equal(0)

    // 1. Add consumer //////////////////////////////////////////////////////////
    // Consumer can be added only by the account owner
    const prepaymentContractNonOwnerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContract.address,
      nonOwnerAddress
    )
    await expect(
      prepaymentContractNonOwnerSigner.addConsumer(accId, consumerAddress)
    ).to.be.revertedWithCustomError(prepaymentContractNonOwnerSigner, 'MustBeAccountOwner')

    // Add consumer with correct signer and parameters
    const txReceiptAddConsumer = await (
      await prepaymentContract.addConsumer(accId, consumerAddress)
    ).wait()
    expect(txReceiptAddConsumer.events.length).to.be.equal(1)

    const accountConsumerAddedEvent = prepaymentContract.interface.parseLog(
      txReceiptAddConsumer.events[0]
    )
    expect(accountConsumerAddedEvent.name).to.be.equal('AccountConsumerAdded')
    const { accId: consumerAddedAccId, consumer: consumerAddedConsumer } =
      accountConsumerAddedEvent.args
    expect(consumerAddedAccId).to.be.equal(accId)
    expect(consumerAddedConsumer).to.be.equal(consumerAddress)

    // Consumers can be access directly through `Account.getConsumers`.
    // Now, there should be single consumer.
    expect((await accountContract.getConsumers()).length).to.be.equal(1)

    // 2. Remove consumer ///////////////////////////////////////////////////////
    // We cannot remove consumer which is not there
    await expect(
      prepaymentContract.removeConsumer(accId, unusedConsumer)
    ).to.be.revertedWithCustomError(accountContract, 'InvalidConsumer')

    // Consumer can be removed only by the account owner
    await expect(
      prepaymentContractNonOwnerSigner.removeConsumer(accId, consumerAddress)
    ).to.be.revertedWithCustomError(prepaymentContractNonOwnerSigner, 'MustBeAccountOwner')

    // Remove consumer with correct signer and paramters
    const txReceiptRemoveConsumer = await (
      await prepaymentContract.removeConsumer(accId, consumerAddress)
    ).wait()
    expect(txReceiptRemoveConsumer.events.length).to.be.equal(1)

    const accountConsumerRemovedEvent = prepaymentContract.interface.parseLog(
      txReceiptRemoveConsumer.events[0]
    )
    expect(accountConsumerRemovedEvent.name).to.be.equal('AccountConsumerRemoved')
    const { accId: consumerRemovedAccId, consumer: consumerRemovedConsumer } =
      accountConsumerRemovedEvent.args

    expect(consumerRemovedAccId).to.be.equal(accId)
    expect(consumerRemovedConsumer).to.be.equal(consumerAddress)

    // After removing the consumer, there should no consumer anymore
    expect((await accountContract.getConsumers()).length).to.be.equal(0)
  })

  it('Add & remove coordinator', async function () {
    const {
      prepaymentContractConsumerSigner,
      prepaymentContract,
      consumer: accountOwnerAddress,
      consumer1: coordinatorAddress
      // consumer1: nonOwnerAddress,
      // consumer2: unusedConsumer
    } = await loadFixture(deployPrepayment)

    // Add coordinator //////////////////////////////////////////////////////////
    // Coordinator must be added by contract owner
    await expect(prepaymentContractConsumerSigner.addCoordinator(coordinatorAddress)).to.be.rejected

    const txAddCoordinator = await (
      await prepaymentContract.addCoordinator(coordinatorAddress)
    ).wait()

    // Check the event information
    expect(txAddCoordinator.events.length).to.be.equal(1)
    const addCoordinatorEvent = prepaymentContract.interface.parseLog(txAddCoordinator.events[0])
    expect(addCoordinatorEvent.name).to.be.equal('CoordinatorAdded')
    const { coordinator: addedCoordinatorAddress } = addCoordinatorEvent.args
    expect(addedCoordinatorAddress).to.be.equal(coordinatorAddress)

    // The same coordinator cannot be added more than once
    await expect(prepaymentContract.addCoordinator(coordinatorAddress)).to.be.rejectedWith(
      'CoordinatorExists'
    )

    // Remove coordinator ///////////////////////////////////////////////////////
    // Non-existing coordinator cannot be removed
    await expect(prepaymentContract.removeCoordinator(NULL_ADDRESS)).to.be.rejectedWith(
      'InvalidCoordinator'
    )

    // We can remove coordinator that has been previously added
    const txRemoveCoordinator = await (
      await prepaymentContract.removeCoordinator(coordinatorAddress)
    ).wait()
    expect((await prepaymentContract.getCoordinators()).length).to.be.equal(0)

    // Check the event information
    expect(txRemoveCoordinator.events.length).to.be.equal(1)
    const removeCoordinatorEvent = prepaymentContract.interface.parseLog(
      txRemoveCoordinator.events[0]
    )
    expect(removeCoordinatorEvent.name).to.be.equal('CoordinatorRemoved')
    const { coordinator: removeCoordinatorAddress } = removeCoordinatorEvent.args
    expect(removeCoordinatorAddress).to.be.equal(coordinatorAddress)
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
    const { accId, account } = accountCreatedEvent.args
    const accountContract = await ethers.getContractAt('Account', account, fromConsumer)

    // 1. Request Account Transfer
    const requestTxReceipt = await (
      await prepaymentContractConsumerSigner.requestAccountOwnerTransfer(accId, toConsumer)
    ).wait()
    expect(requestTxReceipt.events.length).to.be.equal(1)

    const accountTransferRequestedEvent = prepaymentContractConsumerSigner.interface.parseLog(
      requestTxReceipt.events[0]
    )
    expect(accountTransferRequestedEvent.name).to.be.equal('AccountOwnerTransferRequested')
    const { from: fromRequested, to: toRequested } = accountTransferRequestedEvent.args
    expect(fromRequested).to.be.equal(fromConsumer)
    expect(toRequested).to.be.equal(toConsumer)

    expect(await accountContract.getOwner()).to.be.equal(fromConsumer)
    expect(await accountContract.getRequestedOwner()).to.be.equal(toConsumer)

    // 2.1 Cannot accept with wrong consumer
    await expect(
      prepaymentContractConsumerSigner.acceptAccountOwnerTransfer(accId)
    ).to.be.revertedWithCustomError(accountContract, 'MustBeRequestedOwner')

    // 2. Accept Account Transfer
    const prepaymentToConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      prepaymentContractConsumerSigner.address,
      toConsumer
    )
    const acceptTxReceipt = await (
      await prepaymentToConsumerSigner.acceptAccountOwnerTransfer(accId)
    ).wait()
    expect(acceptTxReceipt.events.length).to.be.equal(1)
    const accountTransferredEvent = prepaymentToConsumerSigner.interface.parseLog(
      acceptTxReceipt.events[0]
    )
    expect(accountTransferredEvent.name).to.be.equal('AccountOwnerTransferred')

    const { from: fromTransferred, to: toTransferred } = accountTransferredEvent.args
    expect(fromTransferred).to.be.equal(fromConsumer)
    expect(toTransferred).to.be.equal(toConsumer)

    expect(await accountContract.getOwner()).to.be.equal(toConsumer)
    expect(await accountContract.getRequestedOwner()).to.be.equal(NULL_ADDRESS)
  })
})
