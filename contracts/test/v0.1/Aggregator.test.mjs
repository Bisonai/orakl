import { expect } from 'chai'
import pkg from 'hardhat'
const { ethers } = pkg

let aggregator
const _paymentAmount = 1
const _minSubmissionValue = 2
const _maxSubmissionValue = 3

let owner
let account0
let account1
let account2

async function contractBalance(contract) {
  return await ethers.provider.getBalance(contract)
}

describe('Aggregator', function () {
  beforeEach(async function () {
    ;[owner, account0, account1, account2] = await ethers.getSigners()

    aggregator = await ethers.getContractFactory('Aggregator')

    const _timeout = 10
    const _validator = ethers.constants.AddressZero // no validator
    const _decimals = 18
    const _description = 'Test Aggregator'

    aggregator = await aggregator.deploy(
      _paymentAmount,
      _timeout,
      _validator,
      _minSubmissionValue,
      _maxSubmissionValue,
      _decimals,
      _description
    )
    await aggregator.deployed()

    // Deposit KLAY to Aggregator
    const beforeBalance = await contractBalance(aggregator.address)
    expect(Number(beforeBalance)).to.be.equal(0)
    const value = ethers.utils.parseEther('1.0')
    await aggregator.deposit({ value })
    const afterBalance = await await contractBalance(aggregator.address)
    expect(afterBalance).to.be.equal(value)

    // Register oracles
    const _removed = []
    const _added = [account0.address, account1.address, account2.address]
    const _addedAdmins = [account0.address, account1.address, account2.address]
    const _restartDelay = 0

    await aggregator.changeOracles(
      _removed,
      _added,
      _addedAdmins,
      _minSubmissionValue,
      _maxSubmissionValue,
      _restartDelay
    )
  })

  it('Should accept submissions', async function () {
    // first submission
    const txReceipt0 = await (await aggregator.connect(account0).submit(1, 10)).wait()
    expect(txReceipt0.events[0].event).to.be.equal('NewRound')
    expect(txReceipt0.events[1].event).to.be.equal('SubmissionReceived')
    expect(txReceipt0.events[2].event).to.be.equal('AvailableFundsUpdated')

    // second submission
    const txReceipt1 = await (await aggregator.connect(account1).submit(1, 11)).wait()
    expect(txReceipt1.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt1.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current1 } = txReceipt1.events[1].args
    expect(Number(current1)).to.be.equal(10)
    expect(txReceipt1.events[2].event).to.be.equal('AvailableFundsUpdated')

    // third submission
    const txReceipt2 = await (await aggregator.connect(account2).submit(1, 12)).wait()
    expect(txReceipt2.events[0].event).to.be.equal('SubmissionReceived')
    expect(txReceipt2.events[1].event).to.be.equal('AnswerUpdated')
    const { current: current2 } = txReceipt2.events[1].args
    expect(Number(current2)).to.be.equal(11)
    expect(txReceipt2.events[2].event).to.be.equal('AvailableFundsUpdated')

    const withdrawablePayment0 = await aggregator.withdrawablePayment(account0.address)
    const withdrawablePayment1 = await aggregator.withdrawablePayment(account1.address)
    const withdrawablePayment2 = await aggregator.withdrawablePayment(account2.address)

    expect(Number(withdrawablePayment0)).to.be.equal(_paymentAmount)
    expect(Number(withdrawablePayment1)).to.be.equal(_paymentAmount)
    expect(Number(withdrawablePayment2)).to.be.equal(_paymentAmount)
  })
})
