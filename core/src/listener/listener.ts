// 1. Listen on *multiple* smart contracts for a *single* event type.
// 2. Listen on *multiple* smart contracts for *multiple* event types.

import { parseArgs } from 'node:util'
import { loadJson } from '../utils'
import { WORKER_ANY_API_QUEUE_NAME, WORKER_VRF_QUEUE_NAME } from '../settings'
import { LISTENERS_PATH } from '../load-parameters'
import { Event } from './event'
import { processICNEvent, processVrfEvent } from './processor'
import express from 'express'
import { healthCheck } from '../healthchecker'

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
  console.debug('LISTENERS_PATH', LISTENERS_PATH)
  const listener = loadArgs()
  const listenersConfig = await loadJson(LISTENERS_PATH)
  new Event(
    LISTENERS[listener].queueName,
    LISTENERS[listener].fn,
    listenersConfig[listener]
  ).listen()

  // simple health check, later readness, liveness?
  const server = express()
  server.get('/health-check', (_, res) => {
    res.send(healthCheck())
  })
  // later if you need other port or add new listener, must be changed
  const port = listener == 'VRF' ? 8010 : 8020
  server.listen(port)
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
