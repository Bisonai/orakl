import { expect } from 'chai'
import hre from 'hardhat'

export async function createAccount({
  prepaymentContractAddress,
  consumerContractAddress,
  deposit,
  assignConsumer
}: {
  prepaymentContractAddress
  consumerContractAddress
  deposit?: boolean
  assignConsumer?: boolean
}) {
  const { consumer } = await hre.getNamedAccounts()
  const prepaymentContract = await ethers.getContractAt(
    'Prepayment',
    prepaymentContractAddress,
    consumer
  )

  // CREATE ACCOUNT
  const txReceipt = await (await prepaymentContract.createAccount()).wait()
  expect(txReceipt.events.length).to.be.equal(1)

  const txEvent = prepaymentContract.interface.parseLog(txReceipt.events[0])
  const { accId } = txEvent.args
  expect(accId).to.be.equal(1)

  // DEPOSIT 1 ETHER
  if (deposit) {
    await (
      await prepaymentContract.deposit(accId, { value: ethers.utils.parseUnits('1', 'ether') })
    ).wait()
  }

  // ASSIGN CONSUMER TO ACCOUNT
  if (assignConsumer) {
    await (await prepaymentContract.addConsumer(accId, consumerContractAddress)).wait()
  }

  return accId
}
