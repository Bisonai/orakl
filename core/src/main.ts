import { parseArgs } from 'node:util'
import { buildLogger } from './logger'
import { hookConsoleError } from './utils'

import { buildListener as buildRequestResponseListener } from './listener/request-response'
import { IListeners } from './listener/types'
import { buildListener as buildVrfListener } from './listener/vrf'

import { worker as requestResponseWorker } from './worker/request-response'
import { IWorkers } from './worker/types'
import { worker as vrfWorker } from './worker/vrf'

import { createClient, RedisClientType } from 'redis'
import { OraklError, OraklErrorCode } from './errors'
import { launchHealthCheck } from './health-check'
import { getListeners } from './listener/api'
import { postprocessListeners } from './listener/utils'
import { buildReporter as buildRequestResponseReporter } from './reporter/request-response'
import { IReporters } from './reporter/types'
import { buildReporter as buildVrfReporter } from './reporter/vrf'
import { CHAIN, REDIS_HOST, REDIS_PORT } from './settings'

const LISTENERS: IListeners = {
  VRF: buildVrfListener,
  REQUEST_RESPONSE: buildRequestResponseListener,
}

const WORKERS: IWorkers = {
  VRF: vrfWorker,
  REQUEST_RESPONSE: requestResponseWorker,
}

const REPORTERS: IReporters = {
  VRF: buildVrfReporter,
  REQUEST_RESPONSE: buildRequestResponseReporter,
}

const LOGGER = buildLogger()
const FILE_NAME = import.meta.url

async function startListenerService(service: string, redisClient: RedisClientType) {
  const listenersRawConfig = await getListeners({ service, chain: CHAIN })
  const listenersConfig = postprocessListeners({
    listenersRawConfig,
    service,
    chain: CHAIN,
    logger: LOGGER,
  })

  if (!listenersConfig[service]) {
    LOGGER.error({ name: 'main', file: FILE_NAME, service }, 'service')
    throw new OraklError(OraklErrorCode.UndefinedListenerRequested)
  }

  LISTENERS[service](listenersConfig[service], redisClient, LOGGER)
  LOGGER.info('Listener launched')
}

function startWorkerService(service: string, redisClient: RedisClientType) {
  WORKERS[service](redisClient, LOGGER)
  LOGGER.info('Worker launched')
}

function startReporterService(service: string, redisClient: RedisClientType) {
  REPORTERS[service](redisClient, LOGGER)
  LOGGER.info('Reporter launched')
}

async function main() {
  hookConsoleError(LOGGER)

  const redisClient: RedisClientType = createClient({ url: `redis://${REDIS_HOST}:${REDIS_PORT}` })
  await redisClient.connect()

  const service = loadArgs()

  //   start listener, worker, reporter services
  await startListenerService(service, redisClient)
  startWorkerService(service, redisClient)
  startReporterService(service, redisClient)

  launchHealthCheck()
}

function loadArgs(): string {
  const {
    values: { service },
  } = parseArgs({
    options: {
      service: {
        type: 'string',
      },
    },
  })

  if (!service) {
    throw Error('Missing --service argument.')
  }

  if (
    !Object.keys(LISTENERS).includes(service) ||
    !Object.keys(WORKERS).includes(service) ||
    !Object.keys(REPORTERS).includes(service)
  ) {
    throw Error(`${service} is not supported service.`)
  }

  return service
}

main().catch((e) => {
  LOGGER.error(e)
  process.exitCode = 1
})
