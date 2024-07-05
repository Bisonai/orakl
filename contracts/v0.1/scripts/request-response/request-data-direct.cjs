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

  const callbackGasLimit = 500_000
  const numSubmission = 1
  const refundRecipient = requestResponseConsumerSigner.address
  await requestResponseConsumerSigner.requestDataDirectPaymentUint128(
    callbackGasLimit,
    numSubmission,
    refundRecipient,
    {
      value: ethers.utils.parseEther('1.0'),
    },
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
