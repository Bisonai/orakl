const { task } = require('hardhat/config')
require('@nomicfoundation/hardhat-toolbox')
require('@nomiclabs/hardhat-web3')
require('@nomiclabs/hardhat-ethers')
require('hardhat-deploy')
const dotenv = require('dotenv')

dotenv.config()

const commonConfig = {
  gas: 5_000_000,
  accounts: {
    mnemonic: process.env.MNEMONIC || ''
  }
}

const config = {
  solidity: {
    version: '0.8.16',
    settings: {
      optimizer: {
        enabled: true,
        runs: 1_000
      }
    }
  },

  networks: {
    localhost: {
      gasPrice: 250_000_000_000
    },
    baobab: {
      url: 'https://api.baobab.klaytn.net:8651',
      chainId: 1001,
      ...commonConfig,
      gasPrice: 250_000_000_000
    },
    cypress: {
      url: 'https://public-en-cypress.klaytn.net',
      ...commonConfig,
      gasPrice: 250_000_000_000
    }
  },
  paths: {
    sources: './src'
  },
  namedAccounts: {
    deployer: {
      default: 0
    },
    consumer: {
      default: 1,
      baobab: '0x70997970C51812dc3A010C7d01b50e0d17dc79C8'
    },
    feedOracle0: {
      default: 2,
      baobab: '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC'
    },
    feedOracle1: {
      default: 3,
      baobab: '0x90F79bf6EB2c4f870365E785982E1f101E93b906'
    },
    feedOracle2: {
      default: 4,
      baobab: '0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65'
    },
    consumer1: {
      default: 5
    },
    consumer2: {
      default: 6
    },
    account8: {
      default: 7
    },
    rrOracle0: {
      default: 8
    },
    rrOracle1: {
      default: 9
    },
    rrOracle2: {
      default: 10
    },
    rrOracle3: {
      default: 11
    },
    rrOracle4: {
      default: 12
    },
    rrOracle5: {
      default: 13
    },
    vrfOracle0: {
      default: 18,
      baobab: '0xdd2fd4581271e230360230f9337d5c0430bf44c0'
    }
  }
}

task('address', 'Convert mnemonic to address')
  .addParam('mnemonic', "The account's mnemonic")
  .setAction(async (taskArgs, hre) => {
    const wallet = hre.ethers.Wallet.fromMnemonic(taskArgs.mnemonic)
    console.log(wallet.address)
  })

module.exports = config
