import { logger } from 'ethers'
import { buildLogger } from '../logger'
import { POR_AGGREGATOR_HASH } from '../settings'
import { hookConsoleError } from '../utils'
import { fetchWithAggregator } from './fetcher'
import { reportData } from './reporter'

const LOGGER = buildLogger()
const TIMEOUT = 10000

const _main = async () => {
  hookConsoleError(LOGGER)

  const { value, aggregator } = await fetchWithAggregator({
    aggregatorHash: POR_AGGREGATOR_HASH,
    logger: LOGGER
  })

  logger.info(`Fetched data:${value}`)

  await reportData({ value, aggregator, logger: LOGGER })
}

const main = async () => {
  const timeoutPromise = new Promise((_, reject) =>
    setTimeout(() => reject(new Error('Timeout')), TIMEOUT)
  )
  try {
    await Promise.race([timeoutPromise, _main()])
  } catch (error) {
    if (error instanceof Error && error.message === 'Timeout') {
      throw new Error(`Main function timed out`)
    } else {
      throw error
    }
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
