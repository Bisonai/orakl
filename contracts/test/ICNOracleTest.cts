import { expect } from 'chai'
import axios from 'axios'
import Web3 from 'web3'

import pkg from 'hardhat'
import assert from 'node:assert'
const { ethers } = pkg

// // Function to call external API request and return data
// async function callAPI(url) {
//   let res = await axios({
//     url: url,
//     method: 'get',
//     timeout: 8000,
//     headers: {
//       'Content-Type': 'application/json'
//     }
//   })
//   if (res.status == 200) {
//     console.log(res.status)
//   }
//   return res.data
// }

describe('Fetch data from API using Oracle', function () {
  let ICNOracle
  beforeEach(async function () {
    // Deploy ICN Oracle
    let OracleContract = await ethers.getContractFactory('ICNOracle')
    ICNOracle = await OracleContract.deploy()
    await ICNOracle.deployed()
  })

  // Function for listening to events for Oracle
  it('Should listen for events', async function () {
    const tx = await ICNOracle.createNewJob('https://api.publicapis.org/entries')
    const receipt = await tx.wait()

    for (const event of receipt.events) {
      let url = event.args[1]
      let jobId = event.args[0]
      console.log('URL:', url)
      console.log('JobID: ', jobId)
    }
  })

  //   it('Should update oracle data based on event emittance', async function () {
  //     const tx = await ICNOracle.createNewJob('https://api.publicapis.org/entries')
  //     const receipt = await tx.wait()

  //     for (const event of receipt.events) {
  //       let url = event.args[1]
  //       let jobId = event.args[0]
  //       await callAPI(url).then(async (valu) => {
  //         const data = JSON.stringify(valu.entries[0].Category)
  //         console.log(data.substring(0, 31))
  //         const oracleResponse = ethers.utils.formatBytes32String(data.substring(0, 31))
  //         console.log(oracleResponse)
  //         await ICNOracle.fulfillJob(oracleResponse, jobId)
  //       })
  //     }
  //   })

  //   it('Should fetch data from onChain Oracle', async function () {
  //     const tx = await ICNOracle.createNewJob('https://api.publicapis.org/entries')
  //     const receipt = await tx.wait()
  //     let url
  //     let jobId

  //     for (const event of receipt.events) {
  //       url = event.args[1]
  //       jobId = event.args[0]
  //       await callAPI(url).then(async (valu) => {
  //         const data = JSON.stringify(valu.entries[0].Category)
  //         const oracleResponse = ethers.utils.formatBytes32String(data.substring(0, 31))
  //         await ICNOracle.fulfillJob(oracleResponse, jobId)
  //       })
  //     }

  //     let oracleBytes32Data = await ICNOracle.getData(jobId)
  //     console.log(oracleBytes32Data)
  //     const API_DATA = ethers.utils.parseBytes32String(oracleBytes32Data)
  //     console.log(API_DATA)
  // })
})
