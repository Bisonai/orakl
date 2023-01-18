import { expect } from 'chai'

export async function createAccount(prepaymentContract) {
  const txReceipt = await (await prepaymentContract.createAccount()).wait()
  expect(txReceipt.events.length).to.be.equal(1)

  const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
  const { accId } = txEvent.args
  expect(accId).to.be.equal(1)

  return accId
}
