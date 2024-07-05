const path = require('path')
const fs = require('fs')
const { getFormattedDate, loadJson, storeJson } = require('../utils.cjs')
const { ethers } = require('hardhat')
const { parseArgs } = require('node:util')

async function generateWallet() {
  const wallet = ethers.Wallet.createRandom()
  return wallet
}

const readArgs = async () => {
  const requiredArgs = ['--pairs', '--chain']
  const options = {
    pairs: {
      type: 'string',
    },
    chain: {
      type: 'string',
    },
  }
  const { values } = parseArgs({ requiredArgs, options })
  values['pairs'] = JSON.parse(values['pairs'])

  return values
}

async function main() {
  const { pairs, chain } = await readArgs()

  const baseSource = path.join(__filename, `../../../migration/${chain}/Aggregator/`)
  const aggregatorSource = path.join(__filename, '../dataFeedSample.json')
  const tempFolderPath = path.join(__filename, '../../tmp/')
  const data = await loadJson(aggregatorSource)
  const date = getFormattedDate()
  const walletList = []
  const bulkData = {}

  if (!fs.existsSync(baseSource)) {
    fs.mkdirSync(baseSource, { recursive: true })
  }

  if (!fs.existsSync(tempFolderPath)) {
    fs.mkdirSync(tempFolderPath, { recursive: true })
  }

  bulkData['chain'] = chain
  bulkData['bulk'] = []

  for (const priceFeed of pairs) {
    // setup new wallet
    const wallet = await generateWallet()
    const account = {
      dataFeed: priceFeed,
      address: wallet.address,
      privateKey: wallet.privateKey,
      mnemonic: wallet.mnemonic.phrase,
    }
    walletList.push(account)

    // Config Deploy File
    data.deploy.name = priceFeed
    data.deploy.description = priceFeed
    data.changeOracles.added = [wallet.address]

    console.log(data)

    const storeFilePath = `${baseSource}${date}_${priceFeed}.json`
    storeJson(storeFilePath, JSON.stringify(data, null, 2))

    // Bulk .json File
    bulkData['bulk'].push({
      adapterSource: `https://config.orakl.network/adapter/${chain}/${priceFeed.toLowerCase()}.adapter.json`,
      aggregatorSource: `https://config.orakl.network/aggregator/${chain}/${priceFeed.toLowerCase()}.aggregator.json`,
      reporter: {
        walletAddress: wallet.address,
        walletPrivateKey: wallet.privateKey,
      },
    })
  }

  // store Wallets
  console.log(walletList)
  const storeFilePath = `${tempFolderPath}accountList.json`
  storeJson(storeFilePath, JSON.stringify(walletList, null, 2))

  // store Bulk
  console.log(bulkData)
  const storeBulkJsonFilePath = `${tempFolderPath}bulk.json`
  storeJson(storeBulkJsonFilePath, JSON.stringify(bulkData, null, 2))
  process.exit()
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
