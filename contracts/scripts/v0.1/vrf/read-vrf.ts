import { ethers } from 'hardhat'

async function main() {
  const userContract = await ethers.getContract('VRFConsumerMock')
  console.log('VRFConsumerMock', userContract.address)

  const randomWord = await userContract.s_randomWord()
  console.log(`randomWord ${randomWord.toString()}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
