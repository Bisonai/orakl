import { parseArgs } from 'node:util'
import type { RedisClientType } from 'redis'
import { buildLogger } from '../logger'
import { reporter as dataFeedReporter } from './data-feed'
import { reporter as vrfReporter } from './vrf'
import { reporter as requestResponseReporter } from './request-response'
import { launchHealthCheck } from '../health-check'
import { hookConsoleError } from '../utils'
import { IReporters } from './types'
import { createClient } from 'redis'
import { REDIS_HOST, REDIS_PORT } from '../settings'

const REPORTERS: IReporters = {
  AGGREGATOR: dataFeedReporter,
  VRF: vrfReporter,
  REQUEST_RESPONSE: requestResponseReporter
}

const LOGGER = buildLogger('reporter')

async function main() {
  hookConsoleError(LOGGER)
  const reporter = loadArgs()

  const redisClient: RedisClientType = createClient({ url: `redis://${REDIS_HOST}:${REDIS_PORT}` })
  await redisClient.connect()

  REPORTERS[reporter](redisClient, LOGGER)
  launchHealthCheck()
}

function loadArgs(): string {
  const {
    values: { reporter }
  } = parseArgs({
    options: {
      reporter: {
        type: 'string'
      }
    }
  })

  if (!reporter) {
    throw Error('Missing --reporter argument.')
  }

  if (!Object.keys(REPORTERS).includes(reporter)) {
    throw Error(`${reporter} is not supported reporter.`)
  }

  return reporter
}

main().catch((error) => {
  LOGGER.error(error)
  process.exitCode = 1
})
