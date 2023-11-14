const { getFormattedDate, loadJson, storeJson } = require('../utils.cjs')
const { ethers } = require('hardhat')

const priceFeeds = []

async function generateWallet() {
  const wallet = ethers.Wallet.createRandom()
  return wallet
}

async function main() {
  const baseSource = './migration/cypress/Aggregator/'
  const aggregatorSource = './migration/cypress/Aggregator/20231022211021_JOY-USDT.json'
  const data = await loadJson(aggregatorSource)
  const date = getFormattedDate()

  let walletList = []

  for (let priceFeed of priceFeeds) {
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
    storeJson(storeFilePath, JSON.stringify(data, null, 2)
  }

  // store Wallets
  console.log(walletList)
  const storeFilePath = `${baseSource}accountList.json`
  storeJson(storeFilePath, JSON.stringify(walletList, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
