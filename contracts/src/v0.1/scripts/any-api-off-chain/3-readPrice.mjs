import Web3 from 'web3'
import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let UserContract = await ethers.getContractFactory('ICNMock')
  UserContract = await UserContract.attach('0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512')
  console.log('Loaded Contract Address:', UserContract.address)

  /* const value = await UserContract.value() */
  const value = await UserContract.getValue()
  console.log(`value ${value}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
