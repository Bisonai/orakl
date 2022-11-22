require('dotenv').config()
const { expect } = require('chai')
const { ethers } = require('hardhat')
const https = require('axios')
const Web3 = require('web3')

// Function to call external API request and return data
async function callAPI(url) {
  axios
    .get(url)
    .then((res) => {
      console.log(res)
      return res.data
    })
    .catch((err) => {
      return 'ERROR'
    })
}

describe('Fetch data from API using Oracle', function () {
  let ICNOracle
  beforeEach(async function () {
    // Deploy ICN Oracle
    OracleContract = await ethers.getContractFactory('ICNOracle')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()
  })

  // Function for listening to events for Oracle
  it('Should listen for events', async function () {
    console.log(ICNOracle.address)
    const tx = await ICNOracle.fetchData('https://ethereum.stackexchange.com/')
    const receipt = await tx.wait()

    for (const event of receipt.events) {
      console.log(`Event ${event.event} with args ${event.args}`)
    }
  })
})
