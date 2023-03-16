import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { buildListener as buildAggregatorListener } from './aggregator'
import { buildListener as buildVrfListener } from './vrf'
import { buildListener as buildRequestResponseListener } from './request-response'
import { postprocessListeners, validateListenerConfig } from './utils'
import { OraklError, OraklErrorCode } from '../errors'
import { CHAIN } from '../settings'
import { getListeners } from './api'
import { launchHealthCheck } from '../health-check'
import { hookConsoleError } from '../utils'
import { IListeners } from './types'

const LISTENERS: IListeners = {
  Aggregator: buildAggregatorListener,
  VRF: buildVrfListener,
  RequestResponse: buildRequestResponseListener
}

const FILE_NAME = import.meta.url
const LOGGER = buildLogger('listener')
const SERVICE = 'VRF'

async function main() {
  hookConsoleError(LOGGER)
  const listener = loadArgs()
  const listenersRawConfig = await getListeners({ service: SERVICE, chain: CHAIN })
  if (listenersRawConfig.length == 0) {
    throw new OraklError(
      OraklErrorCode.NoListenerFoundGivenRequirements,
      `service: [${SERVICE}], chain: [${CHAIN}]`
    )
  }
  const listenersConfig = postprocessListeners({ listenersRawConfig })
  const isValid = Object.keys(listenersConfig).map((k) =>
    validateListenerConfig(listenersConfig[k], LOGGER)
  )

  if (!isValid.every((t) => t)) {
    throw new OraklError(OraklErrorCode.InvalidListenerConfig)
  }

  if (!LISTENERS[listener] || !listenersConfig[listener]) {
    LOGGER.error({ name: 'listener:main', file: FILE_NAME, listener }, 'listener')
    throw new OraklError(OraklErrorCode.UndefinedListenerRequested)
  }
  LOGGER.info({ name: 'listener:main', file: FILE_NAME, ...listenersConfig }, 'listenersConfig')

  LISTENERS[listener](listenersConfig[listener], LOGGER)

  launchHealthCheck()
}

function loadArgs(): string {
  const {
    values: { listener }
  } = parseArgs({
    options: {
      listener: {
        type: 'string'
      }
    }
  })

  if (!listener) {
    throw Error('Missing --listener argument.')
  }

  if (!Object.keys(LISTENERS).includes(listener)) {
    throw Error(`${listener} is not supported listener.`)
  }

  return listener
}

main().catch((e) => {
  LOGGER.error(e)
  process.exitCode = 1
})
