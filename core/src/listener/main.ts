import { parseArgs } from 'node:util'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { OraklError, OraklErrorCode } from '../errors'
import { launchHealthCheck } from '../health-check'
import { buildLogger } from '../logger'
import { CHAIN, L2_CHAIN, REDIS_HOST, REDIS_PORT } from '../settings'
import { hookConsoleError } from '../utils'
import { getListeners } from './api'
import { buildListener as buildDataFeedListener } from './data-feed'
import { buildListener as buildL2DataFeedListener } from './data-feed-L2'
import { buildListener as buildRequestResponseListener } from './request-response'
import { IListeners } from './types'
import { postprocessListeners } from './utils'
import { buildListener as buildVrfListener } from './vrf'
import { buildListener as buildVrfL2FulfillListener } from './vrf-L2-fulfill'
import { buildListener as buildVrfL2RequestListener } from './vrf-L2-request'

const LISTENERS: IListeners = {
  DATA_FEED: buildDataFeedListener,
  VRF: buildVrfListener,
  REQUEST_RESPONSE: buildRequestResponseListener,
  DATA_FEED_L2: buildL2DataFeedListener,
  VRF_L2_REQUEST: buildVrfL2RequestListener,
  VRF_L2_FULFILL: buildVrfL2FulfillListener
}

const FILE_NAME = import.meta.url
const LOGGER = buildLogger()

async function main() {
  hookConsoleError(LOGGER)
  const service = loadArgs()
  const chain = service == 'VRF_L2_REQUEST' ? L2_CHAIN : CHAIN
  const listenersRawConfig = await getListeners({ service, chain })
  const listenersConfig = postprocessListeners({
    listenersRawConfig,
    service,
    chain: chain,
    logger: LOGGER
  })

  if (!LISTENERS[service] || !listenersConfig[service]) {
    LOGGER.error({ name: 'listener:main', file: FILE_NAME, service }, 'service')
    throw new OraklError(OraklErrorCode.UndefinedListenerRequested)
  }

  const redisClient: RedisClientType = createClient({ url: `redis://${REDIS_HOST}:${REDIS_PORT}` })

  await redisClient.connect()

  LISTENERS[service](listenersConfig[service], redisClient, LOGGER)
  launchHealthCheck()
  LOGGER.info('Listener launched')
}

function loadArgs(): string {
  const {
    values: { service }
  } = parseArgs({
    options: {
      service: {
        type: 'string'
      }
    }
  })

  if (!service) {
    throw Error('Missing --service argument.')
  }

  if (!Object.keys(LISTENERS).includes(service)) {
    throw Error(`${service} is not supported service.`)
  }

  return service
}

main().catch((e) => {
  LOGGER.error(e)
  process.exitCode = 1
})
