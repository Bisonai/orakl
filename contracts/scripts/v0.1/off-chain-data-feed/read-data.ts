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

  await dataFeedConsumerSigner.getLatestPrice()
  const price = await dataFeedConsumerSigner.s_price()
  const decimals = await dataFeedConsumerSigner.decimals()
  console.log(`Price\t${price}`)
  console.log(`Decimals\t${decimals}`)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
