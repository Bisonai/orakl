const { expect } = require('chai')

function parseRequestCanceled(coordinator, tx) {
  const event = coordinator.interface.parseLog(tx.events[0])
  expect(event.name).to.be.equal('RequestCanceled')
  const { requestId } = event.args
  return { requestId }
}

module.exports = {
  parseRequestCanceled,
}
