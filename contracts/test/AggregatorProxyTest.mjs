import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

import chai from 'chai'

describe('Testing Aggregator Proxy Contract', function () {
  let ICNOracle
  let ICNOracle2
  let ICNOracle3
  let ICNOracle4

  let ICNAggregator
  let ICNAggregatorProxy

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
    let OracleContract = await ethers.getContractFactory('ICNOracleAggregator')
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

    // Deploy Aggregator Proxy Contract
    let AggregatorProxyContract = await ethers.getContractFactory('ICNAggregatorProxy')
    ICNAggregatorProxy = await AggregatorProxyContract.deploy(ICNAggregator.address)
    await ICNAggregatorProxy.deployed()
  })

  it('Should fulfill data through Proxys', async function () {
    let latestRound = await ICNAggregatorProxy.latestRound()
    console.log(Number(latestRound))
    expect(Number(latestRound)).to.be.equal(18446744073709552000)
    expect(await ICNAggregatorProxy.getAnswer(latestRound)).to.be.equal(0)

    // TODO: offChain Price Deviation
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
      ETHUSDPRICE
    )
    await ICNOracle2.fulfillOracleRequest(
      requestIds[1],
      ICNAggregator.address,
      '0xd8c6a442',
      ETHUSDPRICE
    )

    await ICNOracle3.fulfillOracleRequest(
      requestIds[2],
      ICNAggregator.address,
      '0xd8c6a442',
      ETHUSDPRICE
    )

    await ICNOracle4.fulfillOracleRequest(
      requestIds[3],
      ICNAggregator.address,
      '0xd8c6a442',
      ETHUSDPRICE
    )

    latestRound = await ICNAggregatorProxy.latestRound()
    expect(Number(latestRound)).to.be.equal(18446744073709551617)
    let latestTimestamp = await ICNAggregatorProxy.latestTimestamp()
    console.log(latestTimestamp)
    expect(
      Number(await ICNAggregatorProxy.latestAnswer()),
      'Latest Answer is not Accurate'
    ).to.be.equal(17000)
    let roundAnswer = await ICNAggregatorProxy.getAnswer(await ICNAggregatorProxy.latestRound())
    expect(roundAnswer, 'Round Answer Returned is not Accurate').to.be.equal(17000)
  })
})
