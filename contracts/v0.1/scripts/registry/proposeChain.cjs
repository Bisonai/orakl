const { ethers } = require('hardhat')
const hre = require('hardhat')

async function main() {
  const { network } = hre
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

  let registry = await ethers.getContract('Registry')
  registry = await ethers.getContractAt('Registry', registry.address, _consumer)

  const fee = ethers.utils.parseEther('1.0')
  const pChainID = '100001'
  const jsonRpc = 'https://123'
  const endpoint = _consumer.address
  const l1Aggregator = _consumer.address
  const l2Aggregator = _consumer.address

  console.log('Propose chain: ', pChainID, jsonRpc, endpoint, l1Aggregator, l2Aggregator)

  const tx = await (
    await registry.proposeChain(pChainID, jsonRpc, endpoint, l1Aggregator, l2Aggregator, {
      value: fee,
    })
  ).wait()
  console.log('Tx', tx)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
