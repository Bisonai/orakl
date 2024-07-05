const { ethers } = require('hardhat')
const hre = require('hardhat')

async function main() {
  const requestResponseConsumerMock = await ethers.getContract('RequestResponseConsumerMock')
  const { consumer } = await hre.getNamedAccounts()

  const requestResponseConsumerSigner = await ethers.getContractAt(
    'RequestResponseConsumerMock',
    requestResponseConsumerMock.address,
    consumer,
  )

  const accId = 1
  const callbackGasLimit = 500_000
  const numSubmission = 1
  await requestResponseConsumerSigner.requestDataUint128(accId, callbackGasLimit, numSubmission)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
