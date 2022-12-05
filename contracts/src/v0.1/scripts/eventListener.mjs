import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let httpProvider = new ethers.providers.JsonRpcProvider()

  let OracleContract = await ethers.getContractFactory('ICNOracle')
  let ICNOracle = await OracleContract.deploy()
  await ICNOracle.deployed()
  console.log('Deployed ICNOracle Address:', ICNOracle.address)

  let UserContract = await ethers.getContractFactory('ICNMock')
  UserContract = await UserContract.deploy(ICNOracle.address)
  await UserContract.deployed()
  console.log('Deployed User Contract Address:', UserContract.address)

  await UserContract.requestData()

  ICNOracle.on(
    'NewRequest',
    async (requestId, jobId, nonce, callbackAddress, callbackFunctionId, _data) => {
      console.log(_data)

      let stringdata = Web3.utils.hexToString(_data)
      console.log(stringdata)
      let url = stringdata.substring(6)
      console.log(url)
    }
  )
}

main()
