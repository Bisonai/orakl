import { expect } from 'chai'
import pkg from 'hardhat'
const { ethers } = pkg

describe('Testing Aggregator Contract', function () {
  const minimumResponse = 2
  const oracleCount = 5
  const ethUsdPrice = [10, 11, 12, 13, 14]

  let icnOracleAggregator
  let icnAggregator
  let ICNAggregator

  let icnOracles = []
  let oracleAddresses = []
  let jobIds = []
  let requestIds = []

  beforeEach(async function () {
    icnOracleAggregator = await ethers.getContractFactory('ICNOracleAggregator')

    for (let i = 0; i < oracleCount; ++i) {
      icnOracles[i] = await icnOracleAggregator.deploy()
      await icnOracles[i].deployed()
      oracleAddresses.push(icnOracles[i].address)

      jobIds.push(ethers.utils.formatBytes32String(`KLAY-USD-${i}`))
    }

    icnAggregator = await ethers.getContractFactory('ICNAggregator')
    icnAggregator = await icnAggregator.deploy(minimumResponse, oracleAddresses, jobIds)
    await icnAggregator.deployed()
  })

  it('Number of oracles should be same as number of predefined prices', function () {
    expect(ethUsdPrice.length).to.be.equal(oracleCount)
  })

  it('Should Request and Fulfill Data from Oracles declared in Aggregator Contract fulfillments', async function () {
    let latestRound = await icnAggregator.latestRound()
    expect(latestRound).to.be.equal(0)
    expect(await icnAggregator.getAnswer(latestRound)).to.be.equal(0)

    // 1. ICNAggregator: requestRate (NewRound)
    // 2. RequestResponseConsumerBase: sendRequestTo
    // 3. ICNOracleAggregator: createNewRequest (NewRequest)

    const txReceipt = await (await icnAggregator.requestRate()).wait()
    expect(txReceipt.events.length).to.be.equal(5)

    for (let i = 0; i < 4; ++i) {
      const event = icnOracleAggregator.interface.parseLog(txReceipt.events[i])
      expect(event.name).to.be.equal('NewRequest')
      requestIds.push(event.args.requestId)
    }

    const event = icnAggregator.interface.parseLog(txReceipt.events[4])
    expect(event.name).to.be.equal('NewRound')

    for (let i = 0; i < oracleCount; ++i) {
      await icnOracles[i].fulfillOracleRequest(
        requestIds[i],
        icnAggregator.address,
        '0xd8c6a442',
        ethUsdPrice[i]
      )
    }

    latestRound = await icnAggregator.latestRound()
    expect(latestRound).to.be.equal(1)

    // FIXME does not aggregate correctly for odd length arrays
    expect(Number(await icnAggregator.latestAnswer())).to.be.equal(12)
  })
})
