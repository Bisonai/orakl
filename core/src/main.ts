import { parseArgs } from 'node:util'
import { buildLogger } from './logger'
import { hookConsoleError } from './utils'

import { buildListener as buildDataFeedListener } from './listener/data-feed'
import { buildListener as buildL2DataFeedListener } from './listener/data-feed-L2'
import { buildListener as buildRequestResponseListener } from './listener/request-response'
import { buildListener as buildRequestResponseL2FulfillListener } from './listener/request-response-L2-fulfill'
import { buildListener as buildRequestResponseL2RequestListener } from './listener/request-response-L2-request'
import { IListeners } from './listener/types'
import { buildListener as buildVrfListener } from './listener/vrf'
import { buildListener as buildVrfL2FulfillListener } from './listener/vrf-L2-fulfill'
import { buildListener as buildVrfL2RequestListener } from './listener/vrf-L2-request'

import { worker as dataFeedWorker } from './worker/data-feed'
import { worker as l2DataFeedWorker } from './worker/data-feed-L2'
import { worker as requestResponseWorker } from './worker/request-response'
import { worker as l2RequestResponseFulfillWorker } from './worker/request-response-L2-fulfill'
import { worker as l2RequestResponseRequestWorker } from './worker/request-response-L2-request'
import { IWorkers } from './worker/types'
import { worker as vrfWorker } from './worker/vrf'
import { worker as l2VrfFulfillWorker } from './worker/vrf-L2-fulfill'
import { worker as l2VrfRequestWorker } from './worker/vrf-L2-request'

import { createClient, RedisClientType } from 'redis'
import { OraklError, OraklErrorCode } from './errors'
import { launchHealthCheck } from './health-check'
import { getListeners } from './listener/api'
import { postprocessListeners } from './listener/utils'
import { buildReporter as buildDataFeedReporter } from './reporter/data-feed'
import { buildReporter as buildL2DataFeedReporter } from './reporter/data-feed-L2'
import { buildReporter as buildRequestResponseReporter } from './reporter/request-response'
import { buildReporter as buildL2RequestResponseFulfillReporter } from './reporter/request-response-L2-fulfill'
import { buildReporter as buildL2RequestResponseRequestReporter } from './reporter/request-response-L2-request'
import { IReporters } from './reporter/types'
import { buildReporter as buildVrfReporter } from './reporter/vrf'
import { buildReporter as buildL2VrfFulfillReporter } from './reporter/vrf-L2-fulfill'
import { buildReporter as buildL2VrfRequestReporter } from './reporter/vrf-L2-request'
import { CHAIN, REDIS_HOST, REDIS_PORT } from './settings'

const LISTENERS: IListeners = {
  DATA_FEED: buildDataFeedListener,
  VRF: buildVrfListener,
  REQUEST_RESPONSE: buildRequestResponseListener,
  DATA_FEED_L2: buildL2DataFeedListener,
  VRF_L2_REQUEST: buildVrfL2RequestListener,
  VRF_L2_FULFILL: buildVrfL2FulfillListener,
  REQUEST_RESPONSE_L2_REQUEST: buildRequestResponseL2RequestListener,
  REQUEST_RESPONSE_L2_FULFILL: buildRequestResponseL2FulfillListener,
}

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

const REPORTERS: IReporters = {
  DATA_FEED: buildDataFeedReporter,
  VRF: buildVrfReporter,
  REQUEST_RESPONSE: buildRequestResponseReporter,
  DATA_FEED_L2: buildL2DataFeedReporter,
  VRF_L2_REQUEST: buildL2VrfRequestReporter,
  VRF_L2_FULFILL: buildL2VrfFulfillReporter,
  REQUEST_RESPONSE_L2_REQUEST: buildL2RequestResponseRequestReporter,
  REQUEST_RESPONSE_L2_FULFILL: buildL2RequestResponseFulfillReporter,
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
