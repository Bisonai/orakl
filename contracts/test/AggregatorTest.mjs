import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

import chai from 'chai'

function calculateFunctionSelectorId(functionNameWithParameters) {
  return Web3.utils.sha3(functionNameWithParameters).substring(0, 10)
}

describe('Testing Aggregator Contract', function () {
  let ICNOracle
  let ICNOracle2
  let ICNOracle3
  let ICNOracle4

  let ICNAggregator

  let minimumResponse = 2
  let oracleAddresses = []
  let jobIds = [
    ethers.utils.formatBytes32String('JOBID-1-KLAYUSD'),
    ethers.utils.formatBytes32String('JOBID-2-KLAYUSD'),
    ethers.utils.formatBytes32String('JOBID-3-KLAYUSD'),
    ethers.utils.formatBytes32String('JOBID-3-KLAYUSD')
  ]
  let requestIds = []
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

    ICNOracle4 = await OracleContract.deploy()
    await ICNOracle4.deployed()
    oracleAddresses.push(ICNOracle4.address)

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
    let ETHUSDPRICE = 17000
    for (const event of receipt.events) {
      if (event.event == 'Requested') {
        requestIds.push(event.args.id)
      }
    }
    expect(requestIds.length).to.be.equal(4)
    console.log(requestIds)

    await ICNOracle.fulfillOracleRequest(
      requestIds[0],
      ICNAggregator.address,
      '0xd8c6a442',
      ethers.utils.formatBytes32String(ETHUSDPRICE.toString())
    )
    await ICNOracle2.fulfillOracleRequest(
      requestIds[1],
      ICNAggregator.address,
      '0xd8c6a442',
      ethers.utils.formatBytes32String(ETHUSDPRICE.toString())
    )

    await ICNOracle3.fulfillOracleRequest(
      requestIds[2],
      ICNAggregator.address,
      '0xd8c6a442',
      ethers.utils.formatBytes32String(ETHUSDPRICE.toString())
    )

    await ICNOracle4.fulfillOracleRequest(
      requestIds[3],
      ICNAggregator.address,
      '0xd8c6a442',
      ethers.utils.formatBytes32String(ETHUSDPRICE.toString())
    )

    console.log(await ICNAggregator.getAnswersss())
    latestRound = await ICNAggregator.getlatestRound()
    expect(latestRound).to.be.equal(1)
    console.log(await ICNAggregator.getlatestAnswer())
    expect(Number(await ICNAggregator.getlatestAnswer())).to.be.equal(22)
  })
})
