import { parseArgs } from 'node:util'
import { buildAggregatorListener } from './aggregator'
import { buildVrfListener } from './vrf'
import { buildListener as buildRequestResponseListener } from './request-response'
import { validateListenerConfig } from './utils'
import { IcnError, IcnErrorCode } from '../errors'
import { WORKER_REQUEST_RESPONSE_QUEUE_NAME, WORKER_VRF_QUEUE_NAME, DB, CHAIN } from '../settings'
import { getListeners } from '../settings'
import { healthCheck } from '../health-checker'

const LISTENERS = {
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

async function main() {
  const listener = loadArgs()
  const listenersConfig = await getListeners(DB, CHAIN)

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
