import { expect } from 'chai'
import hre, { ethers } from 'hardhat'

async function main() {
  const vrfConsumerMock = await ethers.getContract('VRFConsumerMock')
  const { consumer } = await hre.getNamedAccounts()

  const vrfConsumerSigner = await ethers.getContractAt(
    'VRFConsumerMock',
    vrfConsumerMock.address,
    consumer,
  )
  const vrfCoordinator = await ethers.getContract('VRFCoordinator')
  const keyHash = '0x47ede773ef09e40658e643fe79f8d1a27c0aa6eb7251749b268f829ea49f2024'
  const accId = 2
  const callbackGasLimit = 500_000
  const numWords = 1
  const txReceipt = await (
    await vrfConsumerSigner.requestRandomWords(keyHash, accId, callbackGasLimit, numWords)
  ).wait()
  const event = vrfCoordinator.interface.parseLog(txReceipt.events[0])
  expect(event.args.requestId.toString()).to.not.equal(null)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
