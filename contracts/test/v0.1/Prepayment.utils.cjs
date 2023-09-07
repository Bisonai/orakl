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

async function deposit(prepayment, signer, accId, value) {
  const tx = await (
    await prepayment.connect(signer).deposit(accId, {
      value
    })
  ).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountBalanceIncreased')
  const { accId: eAccId, oldBalance, newBalance } = event.args
  expect(accId).to.be.equal(eAccId)
  return { accId, oldBalance, newBalance }
}

async function withdraw(prepayment, signer, accId, amount) {
  const tx = await (await prepayment.connect(signer).withdraw(accId, amount)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountBalanceDecreased')
  const { accId: eAccId, oldBalance, newBalance } = event.args
  expect(accId).to.be.equal(eAccId)
  return { accId, oldBalance, newBalance }
}

async function addCoordinator(prepayment, signer, coordinatorAddress) {
  const tx = await (await prepayment.connect(signer).addCoordinator(coordinatorAddress)).wait()
  return tx
}

async function cancelAccount(prepayment, signer, accId, to) {
  const tx = await (await prepayment.connect(signer).cancelAccount(accId, to)).wait()
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountCanceled')
  const { accId: eAccId, to: eTo, balance } = event.args
  expect(accId).to.be.equal(eAccId)
  expect(to).to.be.equal(eTo)
  return { accId, to, balance }
}

module.exports = {
  deploy,
  createAccount,
  addConsumer,
  deposit,
  withdraw,
  addCoordinator,
  cancelAccount
}
