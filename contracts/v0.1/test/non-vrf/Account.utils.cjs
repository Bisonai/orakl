const { expect } = require('chai')
const AccountType = {
  TEMPORARY: 0,
  FIAT_SUBSCRIPTION: 1,
  KLAY_SUBSCRIPTION: 2,
  KLAY_DISCOUNT: 3,
  KLAY_REGULAR: 4,
}
function parseAccountCreatedTx(prepayment, tx) {
  expect(tx.events.length).to.be.equal(1)
  const event = prepayment.contract.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('AccountCreated')
  const { accId, account, owner, accType } = event.args
  return { accId, account, owner, accType }
}

module.exports = { parseAccountCreatedTx, AccountType }
