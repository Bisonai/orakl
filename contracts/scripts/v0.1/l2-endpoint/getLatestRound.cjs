const { ethers } = require('hardhat')
const hre = require('hardhat')

async function main() {
  const { network } = hre
  let _consumer

  if (network.name == 'localhost') {
    const { consumer, deployer } = await hre.getNamedAccounts()
    _consumer = consumer
  } else {
    const PROVIDER = process.env.PROVIDER
    const MNEMONIC = process.env.MNEMONIC || ''
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
    _consumer = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
  }
  let aggregator = await ethers.getContract('Aggregator_BNB-USDT_20231018141007')
  aggregator = await ethers.getContractAt('Aggregator', aggregator.address, _consumer)

  const roundData = await aggregator.latestRoundData()
  console.log('Tx', Number(roundData[0]), 'answer', Number(roundData[1]))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
