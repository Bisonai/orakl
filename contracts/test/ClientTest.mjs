import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

describe.skip('ICN Client Contract', function () {
  let UserContract
  let ICNOracle

  it('Should request data from specific requestId of Oracle', async function () {
    let OracleContract = await ethers.getContractFactory('ICNOracle')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()

    let UserContract = await ethers.getContractFactory('ICNMock')
    UserContract = await UserContract.deploy(ICNOracle.address)
    await UserContract.deployed()

    await expect(UserContract.requestData()).to.not.be.reverted
  })

  it('Should recieve an off-chain event of Requested', async function () {
    let OracleContract = await ethers.getContractFactory('ICNOracle')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()

    let UserContract = await ethers.getContractFactory('ICNMock')
    UserContract = await UserContract.deploy(ICNOracle.address)
    await UserContract.deployed()

    const tx = await UserContract.requestData()
    const receipt = await tx.wait()

    for (const event of receipt.events) {
      if (event.event == 'Requested') {
        let requestId = event.args.id
        expect(requestId).to.not.empty
      }
    }
  })
})
