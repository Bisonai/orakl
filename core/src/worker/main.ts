import { parseArgs } from 'node:util'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { IWorkers } from './types'
import { worker as dataFeedWorker } from './data-feed'
import { worker as requestResponseWorker } from './request-response'
import { worker as vrfWorker } from './vrf'
import { buildLogger } from '../logger'
import { launchHealthCheck } from '../health-check'
import { hookConsoleError } from '../utils'
import { REDIS_HOST, REDIS_PORT } from '../settings'

const WORKERS: IWorkers = {
  DATA_FEED: dataFeedWorker,
  VRF: vrfWorker,
  REQUEST_RESPONSE: requestResponseWorker
}

const LOGGER = buildLogger('worker')

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
    values: { worker }
  } = parseArgs({
    options: {
      worker: {
        type: 'string'
      }
    }
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
