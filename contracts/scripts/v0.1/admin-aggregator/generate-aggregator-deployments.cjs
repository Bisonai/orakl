const { getFormattedDate, loadJson, storeJson } = require('../utils.cjs')
const { ethers } = require('hardhat')

// call from contracts workspace
// node ./scripts/v0.1/admin-aggregator/generate-aggregator-deployments.cjs --pairs '["usd-krw", "jpy-usd", "joy-usdc"]' --chain baobab

async function generateWallet() {
  const wallet = ethers.Wallet.createRandom()
  return wallet
}

const readArgs = async () => {
  const requiredArgs = ['--pairs', '--chain']
  const args = process.argv.slice(2)

  if (args.length != 4) {
    throw 'wrong argument numbers, pairs and chain required'
  }

  const result = {}

  for (let i = 0; i < args.length; i += 2) {
    const paramName = args[i]
    const param = args[i + 1]
    if (!requiredArgs.includes(paramName)) {
      throw `wrong argument: ${paramName}, pairs and chain required`
    }

    if (paramName == '--pairs') {
      result['pairs'] = JSON.parse(param)
    }

    if (paramName == '--chain') {
      result['chain'] = param
    }
  }
  return result
}

async function main() {
  const { pairs, chain } = await readArgs()

  const baseSource = `./migration/${chain}/Aggregator/`
  const aggregatorSource = './scripts/v0.1/admin-aggregator/dataFeedSample.json'
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
  const storeBulkJsonFilePath = `${baseSource}${date}_bulk.json`
  storeJson(storeBulkJsonFilePath, JSON.stringify(bulkData, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
