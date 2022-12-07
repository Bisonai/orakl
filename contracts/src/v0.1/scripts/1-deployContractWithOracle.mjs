import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let OracleContract = await ethers.getContractFactory('ICNOracle')
  let ICNOracle = await OracleContract.deploy()
  await ICNOracle.deployed()
  console.log('Deployed ICNOracle Address:', ICNOracle.address)

  let UserContract = await ethers.getContractFactory('ICNMock')
  UserContract = await UserContract.deploy(ICNOracle.address)
  await UserContract.deployed()
  console.log('Deployed User Contract Address:', UserContract.address)
}

main()
