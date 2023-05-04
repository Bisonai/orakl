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
    hardhat: {
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
    // migrations
    deployer: {
      default: 0
    },
    consumer: {
      default: 1
    },
    // tests
    account0: {
      default: 2
    },
    account1: {
      default: 3
    },
    account2: {
      default: 4
    },
    account3: {
      default: 5
    },
    account4: {
      default: 6
    },
    account5: {
      default: 7
    },
    account6: {
      default: 8
    },
    account7: {
      default: 9
    },
    account8: {
      default: 10
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
