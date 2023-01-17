import { HardhatUserConfig, task } from 'hardhat/config'
import '@nomicfoundation/hardhat-toolbox'
import '@nomiclabs/hardhat-web3'
import '@nomiclabs/hardhat-ethers'
import 'hardhat-deploy'
import dotenv from 'dotenv'

dotenv.config()

const commonConfig = {
  gas: 5_000_000,
  accounts: {
    mnemonic: process.env.MNEMONIC || ''
  }
}

const config: HardhatUserConfig = {
  solidity: {
    version: '0.8.16',
    settings: {
      optimizer: {
        enabled: true,
        runs: 1000
      }
    }
  },

  networks: {
    localhost: {
      gas: 1_400_000,
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
      default: 0,
      baobab: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266'
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
    }
  }
}

task('address', 'Convert mnemonic to address')
  .addParam('mnemonic', "The account's mnemonic")
  .setAction(async (taskArgs, hre) => {
    const wallet = hre.ethers.Wallet.fromMnemonic(taskArgs.mnemonic)
    console.log(wallet.address)
  })

export default config
