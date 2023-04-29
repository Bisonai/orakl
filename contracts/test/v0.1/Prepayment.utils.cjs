const { expect } = require('chai')

async function deploy(protocolFeeRecipientAddress, signer) {
  let contract = await ethers.getContractFactory('Prepayment', {
    signer
  })
  contract = await contract.deploy(protocolFeeRecipientAddress)
  await contract.deployed()
  return contract
}

async function createAccount(prepayment, signer) {
  const tx = await (await prepayment.connect(signer).createAccount()).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountCreated')
  const { accId, account, owner } = event.args
  return { accId, account, owner }
}

async function addConsumer(prepayment, signer, accId, consumerAddress) {
  await prepayment.connect(signer).addConsumer(accId, consumerAddress)
}

async function deposit(prepayment, signer, accId, amount) {
  await prepayment.connect(signer).deposit(accId, {
    value: ethers.utils.parseUnits(amount, 'ether')
  })
}

module.exports = {
  deploy,
  createAccount,
  addConsumer,
  deposit
}
