import { expect } from 'chai'
import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { BigNumber } from 'ethers'

function vrfConfig() {
  const oracle = '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199' // FIXME
  const publicProvingKey = [
    '95162740466861161360090244754314042169116280320223422208903791243647772670481',
    '53113177277038648369733569993581365384831203706597936686768754351087979105423'
  ]

  const minimumRequestConfirmations = 3
  const maxGasLimit = 1_000_000
  const gasAfterPaymentCalculation = 1_000
  const feeConfig = {
    fulfillmentFlatFeeLinkPPMTier1: 0,
    fulfillmentFlatFeeLinkPPMTier2: 0,
    fulfillmentFlatFeeLinkPPMTier3: 0,
    fulfillmentFlatFeeLinkPPMTier4: 0,
    fulfillmentFlatFeeLinkPPMTier5: 0,
    reqsForTier2: 0,
    reqsForTier3: 0,
    reqsForTier4: 0,
    reqsForTier5: 0
  }

  return {
    oracle,
    publicProvingKey,
    minimumRequestConfirmations,
    maxGasLimit,
    gasAfterPaymentCalculation,
    feeConfig
  }
}

function parseEther(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18)
}

async function createAccount(prepayment) {
  const txReceipt = await (await prepayment.createAccount()).wait()
  expect(txReceipt.events.length).to.be.equal(1)

  const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
  const { accId } = txEvent.args
  expect(accId).to.be.equal(1)

  return accId
}

describe('Prepayment contract', function () {
  async function deployFixture() {
    const prepaymentContract = await ethers.getContractFactory('Prepayment')
    const [owner, addr1, addr2] = await ethers.getSigners()
    const prepayment = await prepaymentContract.deploy()
    await prepayment.deployed()

    return { prepayment, owner, addr1, addr2 }
  }

  async function deployMockFixture() {
    const [owner, account0, account1, account2] = await ethers.getSigners()

    const prepaymentContract = await ethers.getContractFactory('Prepayment')
    const prepayment = await prepaymentContract.deploy()

    const coordinatorContract = await ethers.getContractFactory('VRFCoordinator')
    const coordinator = await coordinatorContract.deploy(prepayment.address)

    const consumerContract = await ethers.getContractFactory('VRFConsumerMock')
    const consumer = await consumerContract.deploy(coordinator.address)

    const accId = await createAccount(prepayment)

    return { prepayment, coordinator, consumer, owner, accId }
  }

  it('Should create Account', async function () {
    const { prepayment, owner, addr1, addr2 } = await loadFixture(deployFixture)
    const txReceipt = await (await prepayment.createAccount()).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    expect(accId).to.be.equal(1)
  })

  it('Should add consumer', async function () {
    const { prepayment, owner, addr1, addr2 } = await loadFixture(deployFixture)
    const txReceipt = await (await prepayment.createAccount()).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    expect(accId).to.be.equal(1)

    const ownerOfAccId = await prepayment.getAccountOwner(accId)
    expect(ownerOfAccId).to.be.equal(owner.address)

    await prepayment.addConsumer(accId, '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9')
    await prepayment.addConsumer(accId, '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512')

    const transactionTemp = await prepayment.getAccount(accId)
    expect(transactionTemp.consumers.length).to.equal(2)
  })

  it('Should remove consumer', async function () {
    const { prepayment, owner, addr1, addr2 } = await loadFixture(deployFixture)

    const txReceipt = await (await prepayment.createAccount()).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    expect(accId).to.be.equal(1)

    const consumer0 = '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9'
    const consumer1 = '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'

    await prepayment.addConsumer(accId, consumer0)
    await prepayment.addConsumer(accId, consumer1)

    const consumerBefore = (await prepayment.getAccount(accId)).consumers.length
    await prepayment.removeConsumer(accId, consumer1)
    const consumerAfter = (await prepayment.getAccount(accId)).consumers.length

    expect(consumerBefore).to.be.equal(consumerAfter + 1)
  })

  it('Should deposit', async function () {
    const { prepayment, owner, addr1, addr2 } = await loadFixture(deployFixture)
    const txReceipt = await (await prepayment.createAccount()).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    expect(accId).to.be.equal(1)

    const balanceBefore = await prepayment.getAccount(accId)
    const value = 1_000_000_000_000_000
    await prepayment.deposit(accId, { value })
    const balanceAfter = await prepayment.getAccount(accId)
    expect(balanceBefore.balance + value).to.be.equal(balanceAfter.balance)
  })

  it('Should withdraw', async function () {
    const { prepayment, owner, addr1, addr2 } = await loadFixture(deployFixture)
    const accId = await createAccount(prepayment)

    const depositValue = 100_000
    const transactionDeposit = await prepayment.deposit(accId, { value: depositValue })

    const balanceOwnerBefore = await ethers.provider.getBalance(owner.address)
    const balanceAccBefore = (await prepayment.getAccount(accId)).balance
    expect(balanceAccBefore).to.be.equal(depositValue)

    const withdrawValue = 50_000
    const txReceipt = await (await prepayment.withdraw(accId, withdrawValue)).wait()

    const balanceOwnerAfter = await ethers.provider.getBalance(owner.address)
    const balanceAccAfter = (await prepayment.getAccount(accId)).balance

    expect(balanceAccAfter).to.be.equal(depositValue - withdrawValue)

    // FIXME
    console.log(balanceOwnerBefore - balanceOwnerAfter)
    console.log(txReceipt.effectiveGasPrice.toString())
    // expect(balanceOwnerAfter).to.be.greaterThan(balanceOwnerBefore) // WRONG
  })

  it('Should cancel Account', async function () {
    const { prepayment, owner } = await loadFixture(deployMockFixture)
    const txReceipt = await (await prepayment.cancelAccount(1, owner.address)).wait()
    expect(txReceipt.events.length).to.be.equal(1)

    const txEvent = prepayment.interface.parseLog(txReceipt.events[0])
    const { accId, to, balance } = txEvent.args
    expect(accId).to.be.equal(1)
    expect(to).to.be.equal(owner.address)
    expect(balance).to.be.equal(undefined)
  })

  it('Should not cancel Account with pending tx', async function () {
    const { prepayment, owner, coordinator, consumer, accId } = await loadFixture(deployMockFixture)
    const {
      oracle,
      publicProvingKey,
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    } = vrfConfig()

    await coordinator.registerProvingKey(oracle, publicProvingKey)

    await coordinator.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    )

    await prepayment.addConsumer(accId, consumer.address)
    await prepayment.addCoordinator(coordinator.address)

    await consumer.requestRandomWords()

    await expect(prepayment.cancelAccount(1, owner.address)).to.be.revertedWithCustomError(
      prepayment,
      'PendingRequestExists'
    )
  })

  it('Should remove Coordinator', async function () {
    const { prepayment, owner, coordinator, consumer, accId } = await loadFixture(deployMockFixture)
    const {
      oracle,
      publicProvingKey,
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    } = vrfConfig()

    await coordinator.registerProvingKey(oracle, publicProvingKey)

    await coordinator.setConfig(
      minimumRequestConfirmations,
      maxGasLimit,
      gasAfterPaymentCalculation,
      feeConfig
    )

    await prepayment.addConsumer(accId, consumer.address)
    await prepayment.addCoordinator(coordinator.address)
    const tx = await (await prepayment.removeCoordinator(coordinator.address)).wait()
    expect(tx.status).to.equal(1)
  })
})
