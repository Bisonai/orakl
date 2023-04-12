import { expect } from 'chai'
import { ethers } from 'hardhat'
import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'

describe('Prepayment', function () {
  async function deployPrepayment() {
    const { deployer, consumer, consumer1, consumer2 } = await hre.getNamedAccounts()

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

    return {
      deployer,
      consumer,
      consumer1,
      consumer2,
      prepaymentContract,
      prepaymentContractConsumerSigner
    }
  }

  it('Should add and remove consumer', async function () {
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

  // it('Should add consumer', async function () {
  //   const { prepaymentContractConsumerSigner, consumer, deployer, accId } = await loadFixture(
  //     deployFixture
  //   )
  //
  //   const ownerOfAccId = await prepaymentContractConsumerSigner.getAccountOwner(accId)
  //   expect(ownerOfAccId).to.be.equal(consumer)
  //
  //   await prepaymentContractConsumerSigner.addConsumer(
  //     accId,
  //     '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9'
  //   )
  //   await prepaymentContractConsumerSigner.addConsumer(
  //     accId,
  //     '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'
  //   )
  //
  //   const transactionTemp = await prepaymentContractConsumerSigner.getAccount(accId)
  //   expect(transactionTemp.consumers.length).to.equal(2)
  // })
  //
  // it('Should remove consumer', async function () {
  //   const { prepaymentContractConsumerSigner, deployer, accId } = await loadFixture(deployFixture)
  //   const consumer0 = '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9'
  //   const consumer1 = '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'
  //
  //   await prepaymentContractConsumerSigner.addConsumer(accId, consumer0)
  //   await prepaymentContractConsumerSigner.addConsumer(accId, consumer1)
  //
  //   const consumerBefore = (await prepaymentContractConsumerSigner.getAccount(accId)).consumers
  //     .length
  //   await prepaymentContractConsumerSigner.removeConsumer(accId, consumer1)
  //   const consumerAfter = (await prepaymentContractConsumerSigner.getAccount(accId)).consumers
  //     .length
  //
  //   expect(consumerBefore).to.be.equal(consumerAfter + 1)
  // })
  //
  // it('Should deposit', async function () {
  //   const { prepaymentContractConsumerSigner, accId } = await loadFixture(deployFixture)
  //   const balanceBefore = await prepaymentContractConsumerSigner.getAccount(accId)
  //   const value = 1_000_000_000_000_000
  //   await prepaymentContractConsumerSigner.deposit(accId, { value })
  //   const balanceAfter = await prepaymentContractConsumerSigner.getAccount(accId)
  //   expect(balanceBefore.balance.add(value)).to.be.equal(balanceAfter.balance)
  // })
  //
  // it('Should withdraw', async function () {
  //   const { prepaymentContractConsumerSigner, consumer, accId } = await loadFixture(deployFixture)
  //   const depositValue = 100_000
  //   const transactionDeposit = await prepaymentContractConsumerSigner.deposit(accId, {
  //     value: depositValue
  //   })
  //
  //   const balanceOwnerBefore = await ethers.provider.getBalance(consumer)
  //   const balanceAccBefore = (await prepaymentContractConsumerSigner.getAccount(accId)).balance
  //   expect(balanceAccBefore).to.be.equal(depositValue)
  //
  //   const withdrawValue = 50_000
  //   const txReceipt = await (
  //     await prepaymentContractConsumerSigner.withdraw(accId, withdrawValue)
  //   ).wait()
  //
  //   const balanceOwnerAfter = await ethers.provider.getBalance(consumer)
  //   const balanceAccAfter = (await prepaymentContractConsumerSigner.getAccount(accId)).balance
  //   expect(balanceAccAfter).to.be.equal(depositValue - withdrawValue)
  //
  //   expect(
  //     balanceOwnerBefore
  //       .add(withdrawValue)
  //       .sub(txReceipt.cumulativeGasUsed.mul(txReceipt.effectiveGasPrice))
  //   ).to.be.equal(balanceOwnerAfter)
  // })
  //
  // it('Should cancel Account', async function () {
  //   const { prepaymentContractConsumerSigner, deployer } = await loadFixture(deployFixture)
  //   const txReceipt = await (
  //     await prepaymentContractConsumerSigner.cancelAccount(1, deployer)
  //   ).wait()
  //   expect(txReceipt.events.length).to.be.equal(1)
  //
  //   const txEvent = prepaymentContractConsumerSigner.interface.parseLog(txReceipt.events[0])
  //   const { accId, to, balance } = txEvent.args
  //   expect(accId).to.be.equal(1)
  //   expect(to).to.be.equal(deployer)
  //   expect(balance).to.be.equal(undefined)
  // })
  //
  // it('Should not cancel Account with pending tx', async function () {
  //   const {
  //     accId,
  //     prepaymentContract,
  //     prepaymentContractConsumerSigner,
  //     deployer,
  //     consumer,
  //     coordinatorContract,
  //     consumerContract
  //   } = await loadFixture(deployFixture)
  //   const {
  //     oracle,
  //     publicProvingKey,
  //     keyHash,
  //     maxGasLimit,
  //     gasAfterPaymentCalculation,
  //     feeConfig
  //   } = vrfConfig()
  //
  //   await coordinatorContract.registerOracle(oracle, publicProvingKey)
  //
  //   await coordinatorContract.setConfig(
  //     maxGasLimit,
  //     gasAfterPaymentCalculation,
  //     Object.values(feeConfig)
  //   )
  //
  //   await prepaymentContractConsumerSigner.addConsumer(accId, consumerContract.address)
  //   await prepaymentContract.addCoordinator(coordinatorContract.address)
  //   const value = parseKlay(1)
  //   await prepaymentContractConsumerSigner.deposit(accId, { value })
  //
  //   await consumerContract.requestRandomWords(keyHash, accId, maxGasLimit, 1)
  //
  //   await expect(
  //     prepaymentContractConsumerSigner.cancelAccount(accId, consumer)
  //   ).to.be.revertedWithCustomError(prepaymentContract, 'PendingRequestExists')
  // })
  //
  // it('Should remove Coordinator', async function () {
  //   const {
  //     accId,
  //     prepaymentContract,
  //     prepaymentContractConsumerSigner,
  //     deployer,
  //     coordinatorContract,
  //     consumer
  //   } = await loadFixture(deployFixture)
  //   const {
  //     oracle,
  //     publicProvingKey,
  //     maxGasLimit,
  //     keyHash,
  //     gasAfterPaymentCalculation,
  //     feeConfig
  //   } = vrfConfig()
  //
  //   await coordinatorContract.registerOracle(oracle, publicProvingKey)
  //
  //   await coordinatorContract.setConfig(
  //     maxGasLimit,
  //     gasAfterPaymentCalculation,
  //     Object.values(feeConfig)
  //   )
  //
  //   await prepaymentContractConsumerSigner.addConsumer(accId, consumer)
  //   await prepaymentContract.addCoordinator(coordinatorContract.address)
  //   const txReceipt = await (
  //     await prepaymentContract.removeCoordinator(coordinatorContract.address)
  //   ).wait()
  //   expect(txReceipt.status).to.equal(1)
  // })
  //
  // it('Should chargeFee with burn token', async function () {
  //   const { prepaymentContract, accId } = await loadFixture(deployFixture)
  //   const { feedOracle0: node } = await hre.getNamedAccounts()
  //   const prepaymentNodeSigner = await ethers.getContractAt(
  //     'Prepayment',
  //     prepaymentContract.address,
  //     node
  //   )
  //
  //   const depositValue = 1000
  //   const feeAmount = 109
  //   await prepaymentContract.deposit(accId, { value: depositValue })
  //   await prepaymentContract.addCoordinator(node)
  //
  //   const txReceipt = await (await prepaymentNodeSigner.chargeFee(accId, feeAmount, node)).wait()
  //   const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
  //   const { burnAmount } = txEvent.args
  //   const balanceNode = await prepaymentContract.sNodes(node)
  //   const amount = burnAmount.toNumber() + balanceNode.toNumber()
  //
  //   expect(feeAmount).to.be.equal(amount)
  // })
  // it('Should revert with error invalid coordinator', async function () {
  //   const { prepaymentContract, accId } = await loadFixture(deployFixture)
  //   const { feedOracle0: node } = await hre.getNamedAccounts()
  //   const prepaymentNodeSigner = await ethers.getContractAt(
  //     'Prepayment',
  //     prepaymentContract.address,
  //     node
  //   )
  //   const depositValue = 1000
  //   const feeAmount = 109
  //   await prepaymentContract.deposit(accId, { value: depositValue })
  //
  //   await expect(
  //     prepaymentNodeSigner.chargeFee(accId, feeAmount, node)
  //   ).to.be.revertedWithCustomError(prepaymentContract, 'InvalidCoordinator')
  // })
})
