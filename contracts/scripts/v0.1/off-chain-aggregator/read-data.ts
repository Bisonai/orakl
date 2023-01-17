import { ethers } from 'hardhat'
import hre from 'hardhat'

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

  const dataFeedConsumerMock = await ethers.getContract('DataFeedConsumerMock')
  const dataFeedConsumerSigner = await ethers.getContractAt(
    'DataFeedConsumerMock',
    dataFeedConsumerMock.address
  )

  console.log('DataFeedConsumerMock', dataFeedConsumerMock.address)

  try {
    await dataFeedConsumerSigner.connect(_consumer).getLatestPrice()
    const price = await dataFeedConsumerSigner.s_price()
    const decimals = await dataFeedConsumerSigner.decimals()
    const round = await dataFeedConsumerSigner.s_roundID()
    console.log(`Price\t\t${price}`)
    console.log(`Decimals\t${decimals}`)
    console.log(`Round\t\t${round}`)
  } catch (e) {
    console.log(e)
    console.error('Most likely no submission yet.')
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
