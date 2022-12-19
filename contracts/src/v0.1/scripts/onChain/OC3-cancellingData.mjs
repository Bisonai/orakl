import Web3 from 'web3'
import pkg from 'hardhat'
const { ethers } = pkg
import { expect } from 'chai'

async function main() {
  let httpProvider = new ethers.providers.JsonRpcProvider()

  let OracleContract = await ethers.getContractFactory('ICNOracleAggregator')
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
      console.log(requestId)
      console.log(callbackAddress)
      console.log(callbackFunctionId)
      console.log(_data)

      let stringdata = Web3.utils.hexToString(_data)
      console.log(stringdata)
      let url = stringdata.substring(6)
      console.log(url)

      await ICNOracle.cancelOracleRequest(requestId, callbackAddress, '0xbda71d04')

      // TODO: Parse URL and fetch latest ETH price from API
      await expect(
        ICNOracle.fulfillOracleRequest(
          requestId,
          callbackAddress,
          '0xbda71d04',
          Web3.utils.asciiToHex('2000')
        )
      ).to.be.revertedWith('IncorrectRequest()')
    }
  )
}

main()
