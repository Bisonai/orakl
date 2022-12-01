import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

async function callAPI(url) {
  let res = await axios({
    url: url,
    method: 'get',
    timeout: 8000,
    headers: {
      'Content-Type': 'application/json'
    }
  })
  if (res.status == 200) {
    console.log(res.status)
  }
  return res.data
}

async function main() {
  let httpProvider = new ethers.providers.JsonRpcProvider()

  let OracleContract = await ethers.getContractFactory('ICNOracle')
  let ICNOracle = await OracleContract.deploy()
  await ICNOracle.deployed()

  let privateKey = '0xea6c44ac03bff858b476bba40716402b03e41b8e97e276d1baec7c37d42484a0'
  let wallet = new ethers.Wallet(privateKey, httpProvider)

  let UserContract = await ethers.getContractFactory('ICNMock')
  UserContract = await UserContract.deploy(ICNOracle.address)
  await UserContract.deployed()
  console.log('Deployed User Contract Address:', UserContract.address)

  await UserContract.requestData()

  ICNOracle.on(
    'NewRequest',
    async (_requestId, _nonce, _callbackAddress, _callbackFunctionId, _data) => {
      let stringdata = Web3.utils.hexToString(_data)
      console.log(stringdata)
      console.log(_callbackAddress)
      console.log(_callbackFunctionId)
      let url = stringdata.substring(6)
      console.log(url)
      let APIdata = await callAPI(url)

      console.log(APIdata)

      ICNOracle.APIdata // PARSE string data and send external API Request
    }
  )
}

main()
