import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { buildListener as buildDataFeedListener } from './data-feed'
import { buildListener as buildVrfListener } from './vrf'
import { buildListener as buildRequestResponseListener } from './request-response'
import { postprocessListeners, validateListenerConfig } from './utils'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN } from '../settings'
import { getListeners } from './api'
import { hookConsoleError } from '../utils'
import { IListeners } from './types'

import { createClient } from 'redis'

const LISTENERS /*: IListeners*/ = {
  Aggregator: buildDataFeedListener,
  VRF: buildVrfListener,
  RequestResponse: buildRequestResponseListener
}

const FILE_NAME = import.meta.url
const LOGGER = buildLogger('listener')

async function main() {
  hookConsoleError(LOGGER)
  const service = loadArgs()

  //
  const listenersRawConfig = await getListeners({ service, chain: CHAIN })
  if (listenersRawConfig.length == 0) {
    throw new OraklError(
      OraklErrorCode.NoListenerFoundGivenRequirements,
      `service: [${service}], chain: [${CHAIN}]`
    )
  }
  const listenersConfig = postprocessListeners({ listenersRawConfig })
  const isValid = Object.keys(listenersConfig).map((k) =>
    validateListenerConfig(listenersConfig[k], LOGGER)
  )

  if (!isValid.every((t) => t)) {
    throw new OraklError(OraklErrorCode.InvalidListenerConfig)
  }

  if (!LISTENERS[service] || !listenersConfig[service]) {
    LOGGER.error({ name: 'listener:main', file: FILE_NAME, service }, 'service')
    throw new OraklError(OraklErrorCode.UndefinedListenerRequested)
  }
  LOGGER.info({ name: 'listener:main', file: FILE_NAME, ...listenersConfig }, 'listenersConfig')
  //

  const redisClient = createClient()
  await redisClient.connect()

  LISTENERS[service](listenersConfig[service], redisClient, LOGGER)
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
