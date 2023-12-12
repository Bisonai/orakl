const { getFormattedDate, loadJson, storeJson } = require('../utils.cjs')
const { ethers } = require('hardhat')
const path = require('path')
const { parseArgs } = require('node:util')

// ex. node ./scripts/v0.1/admin-aggregator/generate-aggregator-deployments.cjs --pairs '["usd-krw", "jpy-usd", "joy-usdc"]' --chain baobab

async function generateWallet() {
  const wallet = ethers.Wallet.createRandom()
  return wallet
}

const readArgs = async () => {
  const requiredArgs = ['--pairs', '--chain']
  const options = {
    pairs: {
      type: 'string'
    },
    chain: {
      type: 'string'
    }
  }
  const { values } = parseArgs({ requiredArgs, options })
  values['pairs'] = JSON.parse(values['pairs'])

  return values
}

async function main() {
  const { pairs, chain } = await readArgs()

  const baseSource = path.join(__filename, `../../../../migration/${chain}/Aggregator/`)
  const aggregatorSource = path.join(__filename, '../dataFeedSample.json')
  const data = await loadJson(aggregatorSource)
  const date = getFormattedDate()

  let walletList = []
  const bulkData = {}
  bulkData['chain'] = chain
  bulkData['bulk'] = []

  for (const priceFeed of pairs) {
    // setup new wallet
    const wallet = await generateWallet()
    const account = {
      dataFeed: priceFeed,
      address: wallet.address,
      privateKey: wallet.privateKey,
      mnemonic: wallet.mnemonic.phrase
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
      adapterSource: `https://config.orakl.network/adapter/${priceFeed}.adapter.json`,
      aggregatorSource: `https://config.orakl.network/aggregator/${chain}/${priceFeed}.aggregator.json`,
      reporter: {
        walletAddress: wallet.address,
        walletPrivateKey: wallet.privateKey
      }
    })
  }

  // store Wallets
  console.log(walletList)
  const storeFilePath = `${baseSource}accountList.json`
  storeJson(storeFilePath, JSON.stringify(walletList, null, 2))

  // store Bulk
  console.log(bulkData)
  const storeBulkJsonFilePath = `${baseSource}bulk.json`
  storeJson(storeBulkJsonFilePath, JSON.stringify(bulkData, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
