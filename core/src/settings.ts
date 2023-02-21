import os from 'node:os'
import path from 'node:path'
import sqlite from 'sqlite3'
import { open } from 'sqlite'
import { ethers } from 'ethers'
import { IListenerConfig, IVrfConfig } from './types'
import { aggregatorMapping } from './aggregator'
import { listHandler } from './cli/orakl-cli/src/kv'
import { IcnError, IcnErrorCode } from './errors'
import { mkdir } from './utils'
import * as dotenv from 'dotenv'
dotenv.config()

export const TEST_MIGRATIONS_PATH = 'src/cli/orakl-cli/migrations'
export const DEPLOYMENT_NAME = process.env.DEPLOYMENT_NAME || 'orakl'
export const NODE_ENV = process.env.NODE_ENV
export const HEALTH_CHECK_PORT = process.env.HEALTH_CHECK_PORT
export const CHAIN = process.env.CHAIN || 'localhost'
export const LOG_LEVEL = process.env.LOG_LEVEL || 'info'
export const LOG_DIR = process.env.LOG_DIR || './'

export const ORAKL_DIR = process.env.ORAKL_DIR || path.join(os.homedir(), '.orakl')
export const SETTINGS_DB_FILE = path.join(ORAKL_DIR, 'settings.sqlite')
export const DB = await openDb()

export const PROVIDER_URL = await loadKeyValuePair({ db: DB, key: 'PROVIDER_URL', chain: CHAIN })
export const REDIS_HOST =
  process.env.REDIS_HOST || (await loadKeyValuePair({ db: DB, key: 'REDIS_HOST', chain: CHAIN }))
export const REDIS_PORT = process.env.REDIS_PORT
  ? Number(process.env.REDIS_PORT)
  : Number(await loadKeyValuePair({ db: DB, key: 'REDIS_PORT', chain: CHAIN }))
export const SLACK_WEBHOOK_URL =
  process.env.SLACK_WEBHOOK_URL ||
  (await loadKeyValuePair({ db: DB, key: 'SLACK_WEBHOOK_URL', chain: CHAIN }))
export const PRIVATE_KEY = await loadKeyValuePair({ db: DB, key: 'PRIVATE_KEY', chain: CHAIN })
export const PUBLIC_KEY = await loadKeyValuePair({ db: DB, key: 'PUBLIC_KEY', chain: CHAIN })
export const LOCAL_AGGREGATOR = await loadKeyValuePair({
  db: DB,
  key: 'LOCAL_AGGREGATOR',
  chain: CHAIN
})
export const LISTENER_DELAY = Number(
  await loadKeyValuePair({
    db: DB,
    key: 'LISTENER_DELAY',
    chain: CHAIN
  })
)

// BullMQ
export const REMOVE_ON_COMPLETE = 500
export const REMOVE_ON_FAIL = 1_000

// FIXME Move to Redis
export const LISTENER_ROOT_DIR = './tmp/listener/'

export const localAggregatorFn = aggregatorMapping[LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']
export const FIXED_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-fixed-heartbeat-queue`
export const RANDOM_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-random-heartbeat-queue`
export const WORKER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-queue`
export const WORKER_PREDEFINED_FEED_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-predefined-feed-queue`
export const WORKER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-queue`
export const WORKER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-aggregator-queue`
export const REPORTER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-queue`
export const REPORTER_PREDEFINED_FEED_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-predefined-feed-queue`
export const REPORTER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-queue`
export const REPORTER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-aggregator-queue`

export const ALL_QUEUES = [
  FIXED_HEARTBEAT_QUEUE_NAME,
  RANDOM_HEARTBEAT_QUEUE_NAME,
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  WORKER_PREDEFINED_FEED_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  WORKER_AGGREGATOR_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
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

function createJsonRpcProvider() {
  return new ethers.providers.JsonRpcProvider(PROVIDER_URL)
}

export const PROVIDER = createJsonRpcProvider()

async function openDb() {
  mkdir(path.dirname(SETTINGS_DB_FILE))

  const db = await open({
    filename: SETTINGS_DB_FILE,
    driver: sqlite.Database
  })

  const { count } = await db.get('SELECT count(*) AS count FROM sqlite_master WHERE type="table"')
  if (count == 0) {
    await db.migrate()
  }

  return db
}

export async function loadKeyValuePair({ db, key, chain }: { db; key: string; chain: string }) {
  const kv = await listHandler(db)({ key, chain })

  if (kv.length == 0) {
    throw new IcnError(IcnErrorCode.MissingKeyValuePair, `key: ${key}, chain: ${chain}`)
  } else if (kv.length > 1) {
    throw new IcnError(IcnErrorCode.UnexpectedQueryOutput)
  }

  return kv[0].value as string
}

export function postprocessListeners(listeners): IListenerConfig[] {
  const postprocessed = listeners.reduce((groups, item) => {
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

export async function getListeners(db, chain: string): Promise<IListenerConfig[]> {
  const query = `SELECT Service.name, address, eventName FROM Listener
    INNER JOIN Service ON Service.id = Listener.serviceId
    INNER JOIN Chain ON Chain.id=Listener.chainId AND Chain.name='${chain}'`
  const result = await db.all(query)
  const listeners = postprocessListeners(result)
  return listeners
}

export async function getVrfConfig(db, chain: string): Promise<IVrfConfig> {
  const query = `SELECT sk, pk, pk_x, pk_y FROM VrfKey
    INNER JOIN Chain ON Chain.id = VrfKey.chainId AND Chain.name='${chain}'`
  const vrfConfig = await db.get(query)
  return vrfConfig
}

export async function getAdapters(db, chain: string) {
  const query = `SELECT data FROM Adapter
    INNER JOIN Chain ON Chain.id = Adapter.chainId AND Chain.name='${chain}'`
  const adapters = await db.all(query)
  return adapters
}

export async function getAggregators(db, chain: string) {
  const query = `SELECT data FROM Aggregator
    INNER JOIN Chain ON Chain.id = Aggregator.chainId AND Chain.name='${chain}'`
  const aggregators = await db.all(query)
  return aggregators
}
