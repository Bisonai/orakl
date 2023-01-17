import { expect } from 'chai'
import { loadFixture } from '@nomicfoundation/hardhat-network-helpers'
import { BigNumber } from 'ethers'
import hre from 'hardhat'

function vrfConfig() {
  // FIXME
  const oracle = '0x8626f6940E2eb28930eFb4CeF49B2d1F2C9C1199'
  // FIXME
  const publicProvingKey = [
    '95162740466861161360090244754314042169116280320223422208903791243647772670481',
    '53113177277038648369733569993581365384831203706597936686768754351087979105423'
  ]
  const keyHash = '0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
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
    keyHash,
    gasAfterPaymentCalculation,
    feeConfig
  }
}

function parseEther(amount) {
  return ethers.utils.parseUnits(amount.toString(), 18)
}

async function createAccount(prepaymentContract) {
  const txReceipt = await (await prepaymentContract.createAccount()).wait()
  expect(txReceipt.events.length).to.be.equal(1)

  const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
  const { accId } = txEvent.args
  expect(accId).to.be.equal(1)

  return accId
}

describe('Prepayment contract', function () {
  async function deployFixture() {
    const { deployer, consumer } = await hre.getNamedAccounts()

    let prepaymentContract = await ethers.getContractFactory('Prepayment', { signer: deployer })
    prepaymentContract = await prepaymentContract.deploy()
    await prepaymentContract.deployed()

    return { prepaymentContract, deployer }
  }

  async function deployMockFixture() {
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

  it('Should create Account', async function () {
    const { prepaymentContract } = await loadFixture(deployFixture)
    await createAccount(prepaymentContract)
  })

  it('Should add consumer', async function () {
    const { prepaymentContract, deployer } = await loadFixture(deployFixture)
    const accId = await createAccount(prepaymentContract)

    const ownerOfAccId = await prepaymentContract.getAccountOwner(accId)
    expect(ownerOfAccId).to.be.equal(deployer)

    await prepaymentContract.addConsumer(accId, '0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9')
    await prepaymentContract.addConsumer(accId, '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512')

    const transactionTemp = await prepaymentContract.getAccount(accId)
    expect(transactionTemp.consumers.length).to.equal(2)
  })

  it('Should remove consumer', async function () {
    const { prepaymentContract, deployer } = await loadFixture(deployFixture)
    const accId = await createAccount(prepaymentContract)

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
    const { prepaymentContract } = await loadFixture(deployFixture)
    const accId = await createAccount(prepaymentContract)

    const balanceBefore = await prepaymentContract.getAccount(accId)
    const value = 1_000_000_000_000_000
    await prepaymentContract.deposit(accId, { value })
    const balanceAfter = await prepaymentContract.getAccount(accId)
    expect(balanceBefore.balance + value).to.be.equal(balanceAfter.balance)
  })

  it('Should withdraw', async function () {
    const { prepaymentContract, deployer } = await loadFixture(deployFixture)
    const accId = await createAccount(prepaymentContract)

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
    const { prepaymentContract, deployer } = await loadFixture(deployMockFixture)
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
      await loadFixture(deployMockFixture)
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
      await loadFixture(deployMockFixture)
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
})
