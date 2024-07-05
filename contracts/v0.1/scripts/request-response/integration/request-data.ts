import { expect } from 'chai'
import hre, { ethers } from 'hardhat'

async function main() {
  const requestResponseConsumerMock = await ethers.getContract('RequestResponseConsumerMock')
  const { consumer } = await hre.getNamedAccounts()

  const requestResponseConsumerSigner = await ethers.getContractAt(
    'RequestResponseConsumerMock',
    requestResponseConsumerMock.address,
    consumer,
  )
  const requestResponse = await ethers.getContract('RequestResponseCoordinator')

  const accId = 1
  const callbackGasLimit = 500_000

  const txReceipt = await (
    await requestResponseConsumerSigner.requestData(accId, callbackGasLimit)
  ).wait()
  const event = requestResponse.interface.parseLog(txReceipt.events[0])
  expect(event.args.requestId.toString()).to.not.equal(null)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
