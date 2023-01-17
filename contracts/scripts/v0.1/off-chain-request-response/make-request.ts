import { ethers } from 'hardhat'

async function main() {
  const userContract = await ethers.getContract('RequestResponseConsumerMock')
  console.log('RequestResponseConsumerMock', userContract.address)

  await userContract.makeRequest()
  console.log('Data requested')
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
