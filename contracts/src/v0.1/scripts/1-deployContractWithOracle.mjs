import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let OracleContract = await ethers.getContractFactory('ICNOracle')
  let ICNOracle = await OracleContract.deploy()
  await ICNOracle.deployed()
  console.log('Deployed ICNOracle Address:', ICNOracle.address)
  // Oracle Address - 0x5FbDB2315678afecb367f032d93F642f64180aa3

  let UserContract = await ethers.getContractFactory('ICNMock')
  UserContract = await UserContract.deploy(ICNOracle.address)
  await UserContract.deployed()
  console.log('Deployed User Contract Address:', UserContract.address)
}

main()
