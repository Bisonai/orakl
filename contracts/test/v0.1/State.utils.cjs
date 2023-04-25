class State {
  consumerAddress
  prepaymentContract
  prepaymentContractConsumerSigner
  consumerContract
  coordinatorContract
  accId

  constructor(
    consumerAddress,
    prepaymentContract,
    consumerContract,
    coordinatorContract,
    coordinatorContractOracleSigners
  ) {
    this.consumerAddress = consumerAddress

    this.prepaymentContract = prepaymentContract
    this.consumerContract = consumerContract
    this.coordinatorContract = coordinatorContract
    this.coordinatorContractOracleSigners = coordinatorContractOracleSigners
  }

  async initialize(consumerContractName) {
    this.prepaymentContractConsumerSigner = await ethers.getContractAt(
      'Prepayment',
      this.prepaymentContract.address,
      this.consumerAddress
    )
  }

  async createAccount() {
    const txReceipt = await (await this.prepaymentContractConsumerSigner.createAccount()).wait()
    const txEvent = this.prepaymentContractConsumerSigner.interface.parseLog(txReceipt.events[0])
    const { accId } = txEvent.args
    this.accId = accId

    return this.accId
  }

  async addConsumer(consumerAddress) {
    await this.prepaymentContractConsumerSigner.addConsumer(this.accId, consumerAddress)
  }

  async addCoordinator(coordinatorAddress) {
    await this.prepaymentContract.addCoordinator(coordinatorAddress)
  }

  async getBalance() {
    await this.prepaymentContractConsumerSigner.getBalance(this.accId)
  }

  async deposit(amount) {
    // Deposit to [regular] account
    await this.prepaymentContractConsumerSigner.deposit(this.accId, {
      value: ethers.utils.parseUnits(amount, 'ether')
    })
  }
}

module.exports = {
  State
}
