import { ethers } from 'hardhat'

async function main() {
  const requestResponseConsumerMock = await ethers.getContract('RequestResponseConsumerMock')
  const { consumer } = await hre.getNamedAccounts()

  const requestResponseConsumerSigner = await ethers.getContractAt(
    'RequestResponseConsumerMock',
    requestResponseConsumerMock.address,
    consumer
  )

  const accId = 1
  const requestConfirmations = 3
  const callbackGasLimit = 500_000
  await requestResponseConsumerSigner.requestData(accId, requestConfirmations, callbackGasLimit)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
