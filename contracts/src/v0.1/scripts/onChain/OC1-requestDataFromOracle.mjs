import Web3 from 'web3'
import pkg from 'hardhat'
const { ethers } = pkg

async function main() {
  let httpProvider = new ethers.providers.JsonRpcProvider()

  let OracleContract = await ethers.getContractFactory('ICNOracle')
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

      // TODO: Parse URL and fetch latest ETH price from API
      await ICNOracle.fulfillOracleRequest(
        '0x490e8e14c62c900451fcb592a420341af42d1a6a483354efc7fb1a144b212771',
        '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512',
        '0xbda71d04',
        Web3.utils.asciiToHex('2000')
      )
    }
  )
}

main()
