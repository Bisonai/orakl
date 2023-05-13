const { ethers } = require('hardhat')

async function main() {
  const userContract = await ethers.getContract('RequestResponseConsumerMock')
  console.log('RequestResponseConsumerMock', userContract.address)

  console.log('sResponseUint128', (await userContract.sResponseUint128()).toString())
  // console.log('sResponseInt256', await userContract.sResponseInt256())
  // console.log('sResponseBool', await userContract.sResponseBool())
  // console.log('sResponseString', await userContract.sResponseString())
  // console.log('sResponseBytes32', await userContract.sResponseBytes32())
  // console.log('sResponseBytes', await userContract.sResponseBytes())
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
