class Prepayment {
  consumerAddress
  prepaymentContractAddress
  prepaymentContract
  accId

  constructor(
    consumerAddress,
    prepaymentContractAddress
 )
 {
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

  async addConsumer(consumerAddress) {
    await this.prepaymentContract.addConsumer(this.accId, consumerAddress)
  }

  async getBalance() {
    await this.prepaymentContract.getBalance(this.accId)
  }

  async deposit(amount) {
    // Deposit to [regular] account
    await this.prepaymentContract.deposit(this.accId, {
      value: ethers.utils.parseUnits(amount, 'ether')
    })
  }
}

module.exports = {
  Prepayment
}
