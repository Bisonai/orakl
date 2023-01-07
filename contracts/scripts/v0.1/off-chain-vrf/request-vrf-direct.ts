import { ethers } from 'hardhat'
import hre from 'hardhat'

async function main() {
  const vrfCoordinator = await ethers.getContract('VRFCoordinator')
  const vrfConsumerMock = await ethers.getContract('VRFConsumerMock')
  const { consumer } = await hre.getNamedAccounts()

  const vrfConsumerSigner = await ethers.getContractAt(
    'VRFConsumerMock',
    vrfConsumerMock.address,
    consumer
  )

  await vrfConsumerSigner.requestRandomWordsDirect({ value: '100000000000000000' })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
