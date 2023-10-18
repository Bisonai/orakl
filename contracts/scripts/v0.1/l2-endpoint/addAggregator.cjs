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

  let l2Endpoint = await ethers.getContract('L2Endpoint')
  l2Endpoint = await ethers.getContractAt('L2Endpoint', l2Endpoint.address, _consumer)

  console.log('add aggregator: ', aggregator.address)
  const tx = await (await l2Endpoint.addAggregator(aggregator.address)).wait()
  console.log('Tx', tx)

  console.log('add submitter: ', _consumer.address)
  const txsubmitter = await (await l2Endpoint.addSubmitter(_consumer.address)).wait()
  console.log('Tx', txsubmitter)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
