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

task('read-data-feed', 'Read latest data from DataFeedConsumerMock')
  .addParam('pair', 'Price pair (e.g. KLAY-USDT)')
  .setAction(async (taskArgs, hre) => {
    let _consumer
    if (network.name == 'localhost') {
      const { consumer } = await hre.getNamedAccounts()
      _consumer = consumer
    } else {
      const PROVIDER = process.env.PROVIDER
      const MNEMONIC = process.env.MNEMONIC || ''
      const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
      _consumer = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
    }

    const dataFeedConsumerMock = await ethers.getContract(`DataFeedConsumerMock_${taskArgs.pair}`)
    const dataFeedConsumerSigner = await ethers.getContractAt(
      'DataFeedConsumerMock',
      dataFeedConsumerMock.address,
      _consumer
    )
    console.log('DataFeedConsumerMock', dataFeedConsumerMock.address)

    try {
      await dataFeedConsumerSigner.getLatestRoundData()
      const answer = await dataFeedConsumerSigner.sAnswer()
      const decimals = await dataFeedConsumerSigner.decimals()
      const round = await dataFeedConsumerSigner.sId()
      console.log(`Answer\t\t${answer}`)
      console.log(`Decimals\t${decimals}`)
      console.log(`Round\t\t${round}`)
    } catch (e) {
      console.log(e)
      console.error('Most likely no submission yet.')
    }
  })

task('set-burn-fee-ratio', 'Change the burn fee ratio')
  .addParam('ratio', 'New burn fee ratio')
  .addParam('address', 'Prepayment contract address')
  .setAction(async (taskArgs, hre) => {
    let _deployer
    if (network.name == 'localhost') {
      const { deployer } = await hre.getNamedAccounts()
      _deployer = await ethers.getSigner(deployer)
    } else {
      const PROVIDER = process.env.PROVIDER
      const MNEMONIC = process.env.MNEMONIC || ''
      const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
      _deployer = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
    }

    const prepayment = await ethers.getContractAt('Prepayment', taskArgs.address)
    const curBurnFeeRatio = await prepayment.connect(_deployer).getBurnFeeRatio()
    const tx = await (await prepayment.connect(_deployer).setBurnFeeRatio(taskArgs.ratio)).wait()
    console.log('Tx:', tx)

    const newBurnFeeRatio = await prepayment.connect(_deployer).getBurnFeeRatio()
    console.log(`Burn Fee Ratio changed from:${curBurnFeeRatio} to ${newBurnFeeRatio}`)
  })

task('send-klay', 'Send $KLAY from faucet')
  .addParam('address', 'The receiving address')
  .addParam('amount', 'The amount of $KLAY to be send')
  .setAction(async (taskArgs, hre) => {
    let wallet
    const PROVIDER = process.env.PROVIDER || ''
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)

    if (process.env.PRIVATE_KEY) {
      const PRIVATE_KEY = process.env.PRIVATE_KEY || ''
      wallet = await new ethers.Wallet(PRIVATE_KEY, provider)
    } else {
      const MNEMONIC = process.env.MNEMONIC || ''
      wallet = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
    }

    const to = taskArgs.address || ''
    const amount = taskArgs.amount || '0'
    const value = ethers.utils.parseUnits(amount, 'ether')

    console.log(`Transfer ${amount} Klay from ${wallet.address} to ${to}`)

    const tx = {
      from: wallet.address,
      to: taskArgs.address,
      value
    }
    const txReceipt = await (await wallet.sendTransaction(tx)).wait()

    const balance = await provider.getBalance(to)
    const balanceKlay = hre.web3.utils.fromWei(balance.toString(), 'ether')

    console.log(`After balance of account ${to}: ${balanceKlay} Klay`)
    console.log(txReceipt)
  })
module.exports = config
