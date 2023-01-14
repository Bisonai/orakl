import { parseArgs } from 'node:util'
import { buildAggregatorListener } from './aggregator'
import { buildVrfListener } from './vrf'
import { buildAnyApiListener } from './any-api'
import { loadJson } from '../utils'
import { validateListenerConfig } from './utils'
import { IcnError, IcnErrorCode } from '../errors'
import { WORKER_ANY_API_QUEUE_NAME, WORKER_VRF_QUEUE_NAME } from '../settings'
import { LISTENER_CONFIG_FILE } from '../settings'
import { healthCheck } from '../health-checker'

const LISTENERS = {
  AGGREGATOR: {
    queueName: WORKER_VRF_QUEUE_NAME,
    fn: buildAggregatorListener
  },
  VRF: {
    queueName: WORKER_VRF_QUEUE_NAME,
    fn: buildVrfListener
  },
  ANY_API: {
    queueName: WORKER_ANY_API_QUEUE_NAME,
    fn: buildAnyApiListener
  }
}

async function main() {
  console.debug('LISTENER_CONFIG_FILE', LISTENER_CONFIG_FILE)

  const listener = loadArgs()
  const listenersConfig = await loadJson(LISTENER_CONFIG_FILE)

  const isValid = Object.keys(listenersConfig).map((k) =>
    validateListenerConfig(listenersConfig[k])
  )

  if (!isValid) {
    throw new IcnError(IcnErrorCode.InvalidListenerConfig)
  }

  console.log(listenersConfig)

  const queueName = LISTENERS[listener].queueName
  const buildListener = LISTENERS[listener].fn
  const config = listenersConfig[listener]
  buildListener(queueName, config)

  healthCheck()
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

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
