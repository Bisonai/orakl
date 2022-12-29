// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { parseArgs } from 'node:util'
import { loadJson } from '../utils'
import { WORKER_ANY_API_QUEUE_NAME, WORKER_VRF_QUEUE_NAME } from '../settings'
import { LISTENER_CONFIG_FILE } from '../settings'
import { Event } from './event'
import { processICNEvent, processVrfEvent } from './processor'

const LISTENERS = {
  VRF: {
    queueName: WORKER_VRF_QUEUE_NAME,
    fn: processVrfEvent
  },
  ICN: {
    queueName: WORKER_ANY_API_QUEUE_NAME,
    fn: processICNEvent
  }
}

async function main() {
  console.debug('LISTENER_CONFIG_FILE', LISTENER_CONFIG_FILE)

  const listener = loadArgs()
  const listenersConfig = await loadJson(LISTENER_CONFIG_FILE)

  new Event(
    LISTENERS[listener].queueName,
    LISTENERS[listener].fn,
    listenersConfig[listener]
  ).listen()
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
