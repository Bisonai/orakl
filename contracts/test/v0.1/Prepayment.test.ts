import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { expect } from 'chai'
import hre from 'hardhat'
import { vrfConfig } from './VRF.config'
import { createAccount } from './Prepayment.utils'

describe('Prepayment contract', function () {
  async function deployFixture() {
    const { deployer, consumer } = await hre.getNamedAccounts()

    let prepaymentContract = await ethers.getContractFactory('Prepayment', {
      signer: deployer
    })
    prepaymentContract = await prepaymentContract.deploy()

    let coordinatorContract = await ethers.getContractFactory('VRFCoordinator', {
      signer: deployer
    })
    coordinatorContract = await coordinatorContract.deploy(prepaymentContract.address)

    let consumerContract = await ethers.getContractFactory('VRFConsumerMock', {
      signer: consumer
    })
    consumerContract = await consumerContract.deploy(coordinatorContract.address)

    const accId = await createAccount(prepaymentContract)

    return { accId, deployer, consumer, prepaymentContract, coordinatorContract, consumerContract }
  }

  it('Should add consumer', async function () {
    const { prepaymentContract, deployer, accId } = await loadFixture(deployFixture)
    const ownerOfAccId = await prepaymentContract.getAccountOwner(accId)
    expect(ownerOfAccId).to.be.equal(deployer)

    await prepaymentContract.addConsumer(accId, '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9')
    await prepaymentContract.addConsumer(accId, '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512')

    const transactionTemp = await prepaymentContract.getAccount(accId)
    expect(transactionTemp.consumers.length).to.equal(2)
  })

  it('Should remove consumer', async function () {
    const { prepaymentContract, deployer, accId } = await loadFixture(deployFixture)
    const consumer0 = '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9'
    const consumer1 = '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'

    await prepaymentContract.addConsumer(accId, consumer0)
    await prepaymentContract.addConsumer(accId, consumer1)

    const consumerBefore = (await prepaymentContract.getAccount(accId)).consumers.length
    await prepaymentContract.removeConsumer(accId, consumer1)
    const consumerAfter = (await prepaymentContract.getAccount(accId)).consumers.length

    expect(consumerBefore).to.be.equal(consumerAfter + 1)
  })

  it('Should deposit', async function () {
    const { prepaymentContract, accId } = await loadFixture(deployFixture)
    const balanceBefore = await prepaymentContract.getAccount(accId)
    const value = 1_000_000_000_000_000
    await prepaymentContract.deposit(accId, { value })
    const balanceAfter = await prepaymentContract.getAccount(accId)
    expect(balanceBefore.balance + value).to.be.equal(balanceAfter.balance)
  })

  it('Should withdraw', async function () {
    const { prepaymentContract, deployer, accId } = await loadFixture(deployFixture)
    const depositValue = 100_000
    const transactionDeposit = await prepaymentContract.deposit(accId, { value: depositValue })

    const balanceOwnerBefore = await ethers.provider.getBalance(deployer)
    const balanceAccBefore = (await prepaymentContract.getAccount(accId)).balance
    expect(balanceAccBefore).to.be.equal(depositValue)

    const withdrawValue = 50_000
    const txReceipt = await (await prepaymentContract.withdraw(accId, withdrawValue)).wait()

    const balanceOwnerAfter = await ethers.provider.getBalance(deployer)
    const balanceAccAfter = (await prepaymentContract.getAccount(accId)).balance
    expect(balanceAccAfter).to.be.equal(depositValue - withdrawValue)

    expect(
      balanceOwnerBefore
        .add(withdrawValue)
        .sub(txReceipt.cumulativeGasUsed * txReceipt.effectiveGasPrice)
    ).to.be.equal(balanceOwnerAfter)
  })

  it('Should cancel Account', async function () {
    const { prepaymentContract, deployer } = await loadFixture(deployFixture)
    const txReceipt = await (await prepaymentContract.cancelAccount(1, deployer)).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
    const { accId, to, balance } = txEvent.args
    expect(accId).to.be.equal(1)
    expect(to).to.be.equal(deployer)
    expect(balance).to.be.equal(undefined)
  })

  it('Should not cancel Account with pending tx', async function () {
    const { accId, prepaymentContract, deployer, consumer, coordinatorContract, consumerContract } =
      await loadFixture(deployFixture)
    const {
      oracle,
      publicProvingKey,
      minimumRequestConfirmations,
      keyHash,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    } = vrfConfig()

    await coordinatorContract.registerProvingKey(oracle, publicProvingKey)

    await coordinatorContract.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    await prepaymentContract.addConsumer(accId, consumerContract.address)
    await prepaymentContract.addCoordinator(coordinatorContract.address)

    await consumerContract.requestRandomWords(
      keyHash,
      accId,
      minimumRequestConfirmations,
      maxGasLimit,
      1
    )

    await expect(prepaymentContract.cancelAccount(accId, consumer)).to.be.revertedWithCustomError(
      prepaymentContract,
      'PendingRequestExists'
    )
  })

  it('Should remove Coordinator', async function () {
    const { accId, prepaymentContract, deployer, coordinatorContract, consumer } =
      await loadFixture(deployFixture)
    const {
      oracle,
      publicProvingKey,
      minimumRequestConfirmations,
      maxGasLimit,
      keyHash,
      gasAfterPaymentCalculation,
      feeConfig
    } = vrfConfig()

    await coordinatorContract.registerProvingKey(oracle, publicProvingKey)

    await coordinatorContract.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      Object.values(feeConfig)
    )

    await prepaymentContract.addConsumer(accId, consumer)
    await prepaymentContract.addCoordinator(coordinatorContract.address)
    const txReceipt = await (
      await prepaymentContract.removeCoordinator(coordinatorContract.address)
    ).wait()
    expect(txReceipt.status).to.equal(1)
  })

  it('Should chargeFee with burn token', async function () {
    const { prepaymentContract, deployer, accId } = await loadFixture(deployFixture)
    const accounts = await ethers.getSigners()
    const node = accounts[0]

    const depositValue = 1000
    const feeAmount = 109
    const transactionDeposit = await prepaymentContract.deposit(accId, { value: depositValue })
    const role = await prepaymentContract.COORDINATOR_ROLE()
    await prepaymentContract.grantRole(role, node.address)

    const txReceipt = await (
      await prepaymentContract.connect(node).chargeFee(accId, feeAmount, node.address)
    ).wait()
    const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
    const { acc, oldBalance, newBalance, burnAmount } = txEvent.args
    const balanceNode = await prepaymentContract.s_nodes(node.address)
    const amount = burnAmount.toNumber() + balanceNode.toNumber()

    expect(feeAmount).to.be.equal(amount)
    console.log('burn amount', burnAmount.toString(),'- node balance', balanceNode.toString())

  })
})
