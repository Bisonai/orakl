import { logger } from 'ethers'
import { buildLogger } from '../logger'
import { POR_AGGREGATOR_HASH } from '../settings'
import { hookConsoleError } from '../utils'
import { fetchWithAggregator } from './fetcher'
import { reportData } from './reporter'

const aggregatorHash = POR_AGGREGATOR_HASH
const LOGGER = buildLogger()

const main = async () => {
  hookConsoleError(LOGGER)

  const { value, aggregator } = await fetchWithAggregator(aggregatorHash)

  logger.info(`Fetched data:${value}`)

  await reportData({ value, aggregator, logger: LOGGER })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
