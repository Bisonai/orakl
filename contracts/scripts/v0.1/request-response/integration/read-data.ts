import { expect } from 'chai'
import { ethers } from 'hardhat'

async function main() {
  const userContract = await ethers.getContract('RequestResponseConsumerMock')
  console.log('RequestResponseConsumerMock', userContract.address)

  const response = await userContract.s_response()
  console.log(`Response ${response}`)
  expect(response.toString()).to.not.equal('0')
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
