import { ethers } from 'hardhat'
import hre from 'hardhat'

async function main() {
  const { consumer } = await hre.getNamedAccounts()

  const dataFeedConsumerMock = await ethers.getContract('DataFeedConsumerMock')
  const dataFeedConsumerSigner = await ethers.getContractAt(
    'DataFeedConsumerMock',
    dataFeedConsumerMock.address,
    consumer
  )

  console.log('DataFeedConsumerMock', dataFeedConsumerMock.address)

  try {
    await dataFeedConsumerSigner.getLatestPrice()
    const price = await dataFeedConsumerSigner.s_price()
    const decimals = await dataFeedConsumerSigner.decimals()
    const round = await dataFeedConsumerSigner.s_roundID()
    console.log(`Price\t${price}`)
    console.log(`Decimals\t${decimals}`)
    console.log(`Round\t${round}`)
  } catch (e) {
    console.error('Most likely no submission yet.')
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
