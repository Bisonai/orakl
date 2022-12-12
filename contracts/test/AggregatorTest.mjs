import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

describe('Testing Aggregator Contract', function () {
  let ICNOracle
  let ICNOracle2
  let ICNOracle3

  let ICNAggregator

  let minimumResponse = 2
  let oracleAddresses = []
  let jobIds = [
    ethers.utils.formatBytes32String('JOBID-1-KLAYUSD'),
    ethers.utils.formatBytes32String('JOBID-2-KLAYUSD'),
    ethers.utils.formatBytes32String('JOBID-3-KLAYUSD')
  ]
  beforeEach(async function () {
    // Deploy ICN Oracle
    let OracleContract = await ethers.getContractFactory('ICNOracle')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()
    oracleAddresses.push(ICNOracle.address)

    ICNOracle2 = await OracleContract.deploy()
    await ICNOracle2.deployed()
    oracleAddresses.push(ICNOracle2.address)

    ICNOracle3 = await OracleContract.deploy()
    await ICNOracle3.deployed()
    oracleAddresses.push(ICNOracle3.address)

    // Deploy Aggregator Contract

    let AggregatorContract = await ethers.getContractFactory('ICNAggregator')
    ICNAggregator = await AggregatorContract.deploy(minimumResponse, oracleAddresses, jobIds)
    await ICNAggregator.deployed()
    console.log('Aggregator Address: ', ICNAggregator.address)
    console.log('Minimum Responses: ', minimumResponse)
    console.log('Oracle Address:', oracleAddresses)
  })

  it('Should Request Data from Oracles declared in Aggregator Contract', async function () {
    let latestRound = await ICNAggregator.getlatestRound()
    expect(latestRound).to.be.equal(0)
    expect(await ICNAggregator.getAnswer(latestRound)).to.be.equal(0)
    const tx = await ICNAggregator.requestRate()
    const receipt = await tx.wait()
    let requestId
    let ETHUSDPRICE = 1900
    for (const event of receipt.events) {
      if (event.event == 'Requested') {
        requestId = event.args.id
        expect(requestId).to.not.empty
      }
    }

    await ICNAggregator.ICNCallback(requestId, ETHUSDPRICE)
  })
})
