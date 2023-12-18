import { logger } from 'ethers'
import { buildLogger } from '../logger'
import { POR_AGGREGATOR_HASH } from '../settings'
import { hookConsoleError } from '../utils'
import { fetchWithAggregator } from './fetcher'
import { reportData } from './reporter'

const LOGGER = buildLogger()

const main = async () => {
  hookConsoleError(LOGGER)

  const { value, aggregator } = await fetchWithAggregator({
    aggregatorHash: POR_AGGREGATOR_HASH,
    logger: LOGGER
  })

  logger.info(`Fetched data:${value}`)

  await reportData({ value, aggregator, logger: LOGGER })

  return new Promise((_, reject) => {
    setTimeout(() => reject(new Error('Timeout')), 15000) // 15 seconds in milliseconds
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
