import { parseArgs } from 'node:util'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { launchHealthCheck } from '../health-check'
import { buildLogger } from '../logger'
import { REDIS_HOST, REDIS_PORT } from '../settings'
import { hookConsoleError } from '../utils'
import { worker as dataFeedWorker } from './data-feed'
import { worker as l2DataFeedWorker } from './data-feed-L2'
import { worker as requestResponseWorker } from './request-response'
import { worker as l2RequestResponseFulfillWorker } from './request-response-L2-fulfill'
import { worker as l2RequestResponseRequestWorker } from './request-response-L2-request'
import { IWorkers } from './types'
import { worker as vrfWorker } from './vrf'
import { worker as l2VrfFulfillWorker } from './vrf-L2-fulfill'
import { worker as l2VrfRequestWorker } from './vrf-L2-request'

const WORKERS: IWorkers = {
  DATA_FEED: dataFeedWorker,
  VRF: vrfWorker,
  REQUEST_RESPONSE: requestResponseWorker,
  DATA_FEED_L2: l2DataFeedWorker,
  VRF_L2_REQUEST: l2VrfRequestWorker,
  VRF_L2_FULFILL: l2VrfFulfillWorker,
  REQUEST_RESPONSE_L2_REQUEST: l2RequestResponseRequestWorker,
  REQUEST_RESPONSE_L2_FULFILL: l2RequestResponseFulfillWorker,
}

const LOGGER = buildLogger()

async function main() {
  hookConsoleError(LOGGER)
  const worker = loadArgs()

  const redisClient: RedisClientType = createClient({ url: `redis://${REDIS_HOST}:${REDIS_PORT}` })
  await redisClient.connect()

  WORKERS[worker](redisClient, LOGGER)

  LOGGER.info('Worker launched')

  // TODO later replace with watchman after it becomes utilized in every service
  launchHealthCheck()
}

function loadArgs() {
  const {
    values: { worker },
  } = parseArgs({
    options: {
      worker: {
        type: 'string',
      },
    },
  })

  if (!worker) {
    throw Error('Missing --worker argument.')
  }

  if (!Object.keys(WORKERS).includes(worker)) {
    throw Error(`${worker} is not supported worker.`)
  }

  return worker
}

main().catch((e) => {
  LOGGER.error(e)
  process.exitCode = 1
})
