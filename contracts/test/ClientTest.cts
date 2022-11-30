import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

describe('ICN Client Contract', function () {
  let UserContract
  let ICNOracle

  it('Should request data from specific jobId of Oracle', async function () {
    let OracleContract = await ethers.getContractFactory('ICNOracleNew')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()

    let UserContract = await ethers.getContractFactory('ICNMock')
    UserContract = await UserContract.deploy(ICNOracle.address)
    await UserContract.deployed()
    console.log('Deployed User Contract Address:', UserContract.address)

    await UserContract.requestData()
  })
})
