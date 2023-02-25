import { expect } from 'chai'
import { ethers } from 'hardhat'

async function main() {
  const userContract = await ethers.getContract('VRFConsumerMock')

  const randomWord = await userContract.s_randomWord()
  console.log(randomWord.toString())
  expect(randomWord.toString()).to.not.equal('0')
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
