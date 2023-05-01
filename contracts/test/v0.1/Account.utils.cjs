const { expect } = require('chai')

function parseAccountCreatedTx(prepayment, tx) {
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.contract.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountCreated')
  const { accId, account, owner } = event.args
  return { accId, account, owner }
}

module.exports = { parseAccountCreatedTx }
