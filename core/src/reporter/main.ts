import { parseArgs } from 'node:util'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { launchHealthCheck } from '../health-check'
import { buildLogger } from '../logger'
import { REDIS_HOST, REDIS_PORT } from '../settings'
import { hookConsoleError } from '../utils'
import { buildReporter as buildDataFeedReporter } from './data-feed'
import { buildReporter as buildL2DataFeedReporter } from './data-feed-L2'
import { buildReporter as buildRequestResponseReporter } from './request-response'
import { IReporters } from './types'
import { buildReporter as buildVrfReporter } from './vrf'
import { buildReporter as buildVrfL2FulfillReporter } from './vrf-L2-fulfill'
import { buildReporter as buildVrfL2RequestReporter } from './vrf-L2-request'

const REPORTERS: IReporters = {
  DATA_FEED: buildDataFeedReporter,
  VRF: buildVrfReporter,
  REQUEST_RESPONSE: buildRequestResponseReporter,
  DATA_FEED_L2: buildL2DataFeedReporter,
  VRF_L2_REQUEST: buildVrfL2RequestReporter,
  VRF_L2_FULFILL: buildVrfL2FulfillReporter
}

const LOGGER = buildLogger()

async function main() {
  hookConsoleError(LOGGER)
  const reporter = loadArgs()

  const redisClient: RedisClientType = createClient({ url: `redis://${REDIS_HOST}:${REDIS_PORT}` })
  await redisClient.connect()

  REPORTERS[reporter](redisClient, LOGGER)
  launchHealthCheck()

  LOGGER.debug('Reporter launched')
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
