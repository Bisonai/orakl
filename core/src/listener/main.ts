import { parseArgs } from 'node:util'
import { buildLogger } from '../logger'
import { buildAggregatorListener } from './aggregator'
import { buildVrfListener } from './vrf'
import { buildListener as buildRequestResponseListener } from './request-response'
import { validateListenerConfig } from './utils'
import { IcnError, IcnErrorCode } from '../errors'
import { WORKER_REQUEST_RESPONSE_QUEUE_NAME, WORKER_VRF_QUEUE_NAME, DB, CHAIN } from '../settings'
import { getListeners } from '../settings'
import { launchHealthCheck } from '../health-check'
import { hookConsoleError } from '../utils'
import { IListeners } from './types'

const LISTENERS: IListeners = {
  Aggregator: {
    queueName: WORKER_VRF_QUEUE_NAME,
    fn: buildAggregatorListener
  },
  VRF: {
    queueName: WORKER_VRF_QUEUE_NAME,
    fn: buildVrfListener
  },
  RequestResponse: {
    queueName: WORKER_REQUEST_RESPONSE_QUEUE_NAME,
    fn: buildRequestResponseListener
  }
}

const FILE_NAME = import.meta.url
const LOGGER = buildLogger('listener')

async function main() {
  hookConsoleError(LOGGER)
  const listener = loadArgs()
  const listenersConfig = await getListeners(DB, CHAIN)

  const isValid = Object.keys(listenersConfig).map((k) =>
    validateListenerConfig(listenersConfig[k], LOGGER)
  )

  if (!isValid) {
    throw new IcnError(IcnErrorCode.InvalidListenerConfig)
  }

  if (!LISTENERS[listener] || !listenersConfig[listener]) {
    LOGGER.error({ name: 'listener:main', file: FILE_NAME, listener }, 'listener')
    throw new IcnError(IcnErrorCode.UndefinedListenerRequested)
  }

  LOGGER.info({ name: 'listener:main', file: FILE_NAME, ...listenersConfig }, 'listenersConfig')

  const queueName = LISTENERS[listener].queueName
  const buildListener = LISTENERS[listener].fn
  const config = listenersConfig[listener]
  buildListener(queueName, config, LOGGER)

  launchHealthCheck()
}

function loadArgs() {
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
