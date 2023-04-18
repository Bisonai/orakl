import { expect } from 'chai'
import hre from 'hardhat'

export class Prepayment {
  consumerAddress: string
  prepaymentContractAddress: string
  prepaymentContract: ethers.Contract
  accId: number

  constructor({
    consumerAddress,
    prepaymentContractAddress
  }: {
    consumerAddress: string
    prepaymentContractAddress: ethers.Contract
  }) {
    this.consumerAddress = consumerAddress
    this.prepaymentContractAddress = prepaymentContractAddress
  }

  async initialize() {
    this.prepaymentContract = await ethers.getContractAt(
      'Prepayment',
      this.prepaymentContractAddress,
      this.consumerAddress
    )
  }

  async createAccount() {
    const txReceipt = await (await this.prepaymentContract.createAccount()).wait()
    // expect(txReceipt.events.length).to.be.equal(1) // FIXME
    const txEvent = this.prepaymentContract.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    this.accId = accId

    return this.accId
  }

  async addConsumer(consumerAddress: string) {
    await this.prepaymentContract.addConsumer(this.accId, consumerAddress)
  }

  async getBalance() {
    await this.prepaymentContract.getBalance(this.accId)
  }

  async deposit(amount: string) {
    // Deposit to [regular] account
    await this.prepaymentContract.deposit(this.accId, {
      value: ethers.utils.parseUnits(amount, 'ether')
    })
  }
}
