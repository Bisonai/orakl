import * as Path from 'node:path'
import sqlite from 'sqlite3'
import { open } from 'sqlite'
import { aggregatorMapping } from './aggregator'
import { LOCAL_AGGREGATOR, REDIS_HOST, REDIS_PORT } from './load-parameters'

export const localAggregatorFn = aggregatorMapping[LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']

export const FIXED_HEARTBEAT_QUEUE_NAME = 'fixed-heartbeat-queue'

export const RANDOM_HEARTBEAT_QUEUE_NAME = 'random-heartbeat-queue'

export const WORKER_ANY_API_QUEUE_NAME = 'worker-any-api-queue'

export const WORKER_PREDEFINED_FEED_QUEUE_NAME = 'worker-predefined-feed-queue'

export const WORKER_VRF_QUEUE_NAME = 'worker-vrf-queue'

export const WORKER_AGGREGATOR_QUEUE_NAME = 'worker-aggregator-queue'

export const REPORTER_ANY_API_QUEUE_NAME = 'reporter-any-api-queue'

export const REPORTER_PREDEFINED_FEED_QUEUE_NAME = 'reporter-predefined-feed-queue'

export const REPORTER_VRF_QUEUE_NAME = 'reporter-vrf-queue'

export const REPORTER_AGGREGATOR_QUEUE_NAME = 'reporter-aggregator-queue'

export const ALL_QUEUES = [
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  WORKER_ANY_API_QUEUE_NAME,
  WORKER_PREDEFINED_FEED_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_ANY_API_QUEUE_NAME,
  REPORTER_PREDEFINED_FEED_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME
]

export const BULLMQ_CONNECTION = {
  connection: {
    host: REDIS_HOST,
    port: REDIS_PORT
  }
}

export const ADAPTER_ROOT_DIR = './adapter/'

export const AGGREGATOR_ROOT_DIR = './aggregator/'

export const LISTENER_ROOT_DIR = './tmp/listener/'

export const CONFIG_ROOT_DIR = './config/'

export const LISTENER_CONFIG_FILE = Path.join(CONFIG_ROOT_DIR, 'listener.json')

export const VRF_CONFIG_FILE = Path.join(CONFIG_ROOT_DIR, 'vrf.json')

export const LISTENER_DELAY = 500

export const SETTINGS_DB_FILE = './settings.sqlite'

async function openDb() {
  return await open({
    filename: SETTINGS_DB_FILE,
    driver: sqlite.Database
  })
}

export function postprocessListeners(listeners) {
  let postprocessed = listeners.reduce((groups, item) => {
    const group = groups[item.name] || []
    group.push(item)
    groups[item.name] = group
    return groups
  }, {})

  Object.keys(postprocessed).forEach((serviceName) => {
    return postprocessed[serviceName].map((listener) => {
      delete listener['name']
      return listener
    })
  })

  return postprocessed
}

export async function getListeners(chain: string) {
  const db = await openDb()
  const query = `SELECT Service.name, address, eventName FROM Listener
    LEFT OUTER JOIN Service ON Service.id = Listener.serviceId
    LEFT OUTER JOIN Chain ON Chain.id = Listener.chainId AND Chain.name = '${chain}'`
  const result = await db.all(query)

  const listeners = postprocessListeners(result)
  return listeners
}
