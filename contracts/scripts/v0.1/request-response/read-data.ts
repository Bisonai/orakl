import { ethers } from 'hardhat'

async function main() {
  const userContract = await ethers.getContract('RequestResponseConsumerMock')
  console.log('RequestResponseConsumerMock', userContract.address)

  const response = await userContract.sResponse()
  console.log(`Response ${response}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
