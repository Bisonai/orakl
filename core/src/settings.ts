import * as dotenv from 'dotenv'
import { ethers } from 'ethers'
dotenv.config()

export const ORAKL_NETWORK_API_URL =
  process.env.ORAKL_NETWORK_API_URL || 'http://localhost:3000/api/v1'
export const ORAKL_NETWORK_DELEGATOR_URL =
  process.env.ORAKL_NETWORK_DELEGATOR_URL || 'http://localhost:3002/api/v1'

export const DELEGATOR_TIMEOUT = Number(process.env.DELEGATOR_TIMEOUT) || 3000
export const RPC_URL_TIMEOUT = Number(process.env.RPC_URL_TIMEOUT) || 3000
export const POR_TIMEOUT = Number(process.env.POR_TIMEOUT) || 5000

export const DEPLOYMENT_NAME = process.env.DEPLOYMENT_NAME || 'orakl'
export const NODE_ENV = process.env.NODE_ENV
export const HEALTH_CHECK_PORT = process.env.HEALTH_CHECK_PORT
export const CHAIN = process.env.CHAIN || 'localhost'
export const LOG_LEVEL = process.env.LOG_LEVEL || 'info'

export const PROVIDER_URL = process.env.PROVIDER_URL || 'http://127.0.0.1:8545'
export const FALLBACK_PROVIDER_URL = process.env.FALLBACK_PROVIDER_URL
export const REDIS_HOST = process.env.REDIS_HOST || 'localhost'
export const REDIS_PORT = process.env.REDIS_PORT ? Number(process.env.REDIS_PORT) : 6379
export const SLACK_WEBHOOK_URL = process.env.SLACK_WEBHOOK_URL || ''
export const LISTENER_DELAY = Number(process.env.LISTENER_DELAY) || 500

// POR
export const POR_AGGREGATOR_HASH = process.env.POR_AGGREGATOR_HASH || ''
export const POR_LATENCY_BUFFER = 60000 // submission latency buffer for POR in millisecs

// Gas mimimums
export const VRF_FULFILL_GAS_MINIMUM = 1_000_000
export const REQUEST_RESPONSE_FULFILL_GAS_MINIMUM = 400_000
export const DATA_FEED_FULFILL_GAS_MINIMUM = 400_000
export const POR_GAS_MINIMUM = 400_000
export const VRF_FULLFILL_GAS_PER_WORD = 1_000

// Service ports are used for communication to watchman from the outside
export const LISTENER_PORT = process.env.LISTENER_PORT || 4_000
export const WORKER_PORT = process.env.WORKER_PORT || 5_001
export const REPORTER_PORT = process.env.REPORTER_PORT || 6_000

export const DATA_FEED_SERVICE_NAME = 'DATA_FEED'
export const VRF_SERVICE_NAME = 'VRF'
export const REQUEST_RESPONSE_SERVICE_NAME = 'REQUEST_RESPONSE'
export const L2_DATA_FEED_SERVICE_NAME = 'DATA_FEED_L2'
export const L2_VRF_REQUEST_SERVICE_NAME = 'VRF_L2_REQUEST'
export const L2_VRF_FULFILL_SERVICE_NAME = 'VRF_L2_FULFILL'
export const L2_REQUEST_RESPONSE_REQUEST_SERVICE_NAME = 'REQUEST_RESPONSE_L2_REQUEST'
export const L2_REQUEST_RESPONSE_FULFILL_SERVICE_NAME = 'REQUEST_RESPONSE_L2_FULFILL'
export const POR_SERVICE_NAME = 'POR'

// Data Feed
export const MAX_DATA_STALENESS = 5_000

// BullMQ
export const REMOVE_ON_COMPLETE = 500
export const REMOVE_ON_FAIL = 1_000
export const CONCURRENCY = 50
export const DATA_FEED_REPORTER_CONCURRENCY =
  Number(process.env.DATA_FEED_REPORTER_CONCURRENCY) || 15

export const LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-latest-queue`
export const LISTENER_VRF_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-latest-queue`
export const LISTENER_DATA_FEED_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-latest-queue`
export const L2_LISTENER_DATA_FEED_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-latest-queue`
export const L2_LISTENER_VRF_REQUEST_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-request-latest-queue`
export const L2_LISTENER_VRF_FULFILL_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-fulfill-latest-queue`
export const L2_LISTENER_REQUEST_RESPONSE_REQUEST_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-request-latest-queue`
export const L2_LISTENER_REQUEST_RESPONSE_FULFILL_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-fulfill-latest-queue`

export const LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-history-queue`
export const LISTENER_VRF_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-history-queue`
export const LISTENER_DATA_FEED_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-history-queue`
export const L2_LISTENER_DATA_FEED_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-history-queue`
export const L2_LISTENER_VRF_REQUEST_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-request-history-queue`
export const L2_LISTENER_VRF_FULFILL_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-fulfill-history-queue`
export const L2_LISTENER_REQUEST_RESPONSE_REQUEST_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-request-history-queue`
export const L2_LISTENER_REQUEST_RESPONSE_FULFILL_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-fulfill-history-queue`

export const LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-process-event-queue`
export const LISTENER_VRF_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-process-event-queue`
export const LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-process-event-queue`
export const L2_LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-process-event-queue`
export const L2_LISTENER_VRF_REQUEST_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-request-process-event-queue`
export const L2_LISTENER_VRF_FULFILL_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-l2-fulfill-process-event-queue`
export const L2_LISTENER_REQUEST_RESPONSE_REQUEST_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-request-process-event-queue`
export const L2_LISTENER_REQUEST_RESPONSE_FULFILL_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-l2-fulfill-process-event-queue`

export const SUBMIT_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-submitheartbeat-queue`
export const HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-heartbeat-queue`
export const WORKER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-queue`
export const WORKER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-queue`
export const WORKER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-aggregator-queue`
export const WORKER_CHECK_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-checkheartbeat-queue`
export const L2_WORKER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-aggregator-l2-queue`
export const L2_WORKER_VRF_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-request-l2-queue`
export const L2_WORKER_VRF_FULFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-fulfill-l2-queue`
export const L2_WORKER_REQUEST_RESPONSE_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-request-l2-queue`
export const L2_WORKER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-fulfill-l2-queue`

export const NONCE_MANAGER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-request-response-queue`
export const NONCE_MANAGER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-vrf-queue`
export const NONCE_MANAGER_L2_REQUEST_RESPONSE_FUFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-request-response-l2-fulfill-queue`
export const NONCE_MANAGER_L2_REQUEST_RESPONSE_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-request-response-l2-request-queue`
export const NONCE_MANAGER_L2_VRF_FULFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-vrf-l2-fulfill-queue`
export const NONCE_MANAGER_L2_VRF_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-nonce-manager-vrf-l2-request-queue`

export const REPORTER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-queue`
export const REPORTER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-queue`
export const REPORTER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-aggregator-queue`
export const WORKER_DEVIATION_QUEUE_NAME = `orakl-deviation-queue`
export const L2_REPORTER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-aggregator-l2-queue`
export const L2_REPORTER_VRF_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-request-l2-queue`
export const L2_REPORTER_VRF_FULFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-fulfill-l2-queue`
export const L2_REPORTER_REQUEST_RESPONSE_REQUEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-request-l2-queue`
export const L2_REPORTER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-fulfill-l2-queue`

export const HEARTBEAT_JOB_NAME = `${DEPLOYMENT_NAME}-heartbeat-job`

export const L2_CHAIN = process.env.L2_CHAIN || 'localhost'
export const L2_PROVIDER_URL = process.env.L2_PROVIDER_URL || 'http://127.0.0.1:8545'

export const BAOBAB_CHAIN_ID = 1001
export const CYPRESS_CHAIN_ID = 8217

export const ALL_QUEUES = [
  LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME,
  LISTENER_VRF_LATEST_QUEUE_NAME,
  LISTENER_DATA_FEED_LATEST_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME,
  LISTENER_VRF_HISTORY_QUEUE_NAME,
  LISTENER_DATA_FEED_HISTORY_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME,
  LISTENER_VRF_PROCESS_EVENT_QUEUE_NAME,
  LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME,
  SUBMIT_HEARTBEAT_QUEUE_NAME,
  HEARTBEAT_QUEUE_NAME,
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  WORKER_AGGREGATOR_QUEUE_NAME,
  WORKER_CHECK_HEARTBEAT_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  REPORTER_AGGREGATOR_QUEUE_NAME,
  L2_WORKER_AGGREGATOR_QUEUE_NAME
]

export const VRF_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-state`
export const REQUEST_RESPONSE_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-state`
export const DATA_FEED_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-state`
export const L2_DATA_FEED_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-state`
export const L2_VRF_REQUEST_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-request-l2-state`
export const L2_VRF_FULFILL_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-fulfill-l2-state`
export const L2_REQUEST_RESPONSE_REQUEST_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-request-l2-state`
export const L2_REQUEST_RESPONSE_FULFILL_LISTENER_STATE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-fulfill-l2-state`

// export const VRF_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-state`
// export const REQUEST_RESPONSE_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-state`
export const DATA_FEED_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-data-feed-state`
export const L2_DATA_FEED_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-data-feed-l2-state`

export const VRF_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-state`
export const REQUEST_RESPONSE_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-state`
export const DATA_FEED_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-data-feed-state`
export const L2_DATA_FEED_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-data-feed-l2-state`
export const L2_VRF_REQUEST_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-l2-request-state`
export const L2_VRF_FULFILL_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-l2-fulfill-state`
export const L2_REQUEST_RESPONSE_REQUEST_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-l2-request-state`
export const L2_REQUEST_RESPONSE_FULFILL_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-l2-fulfill-state`

export const BULLMQ_CONNECTION = {
  concurrency: CONCURRENCY,
  connection: {
    host: REDIS_HOST,
    port: REDIS_PORT
  }
}

function createJsonRpcProvider(providerUrl: string = PROVIDER_URL) {
  return new ethers.providers.JsonRpcProvider(providerUrl)
}

export const PROVIDER = createJsonRpcProvider()
export const L2_PROVIDER = createJsonRpcProvider(L2_PROVIDER_URL)
export const L1_ENDPOINT = process.env.L1_ENDPOINT || ''
export const L2_ENDPOINT = process.env.L2_ENDPOINT || ''

export const HEARTBEAT_QUEUE_SETTINGS = {
  removeOnComplete: true,
  attempts: 10,
  backoff: 1_000
}

export const AGGREGATOR_QUEUE_SETTINGS = {
  // When [aggregator] worker fails, we want to be able to
  // resubmit the job with the same job ID.
  removeOnFail: true,
  attempts: 10,
  backoff: 1_000
}

export const SUBMIT_HEARTBEAT_QUEUE_SETTINGS = {
  removeOnComplete: REMOVE_ON_COMPLETE,
  removeOnFail: REMOVE_ON_FAIL,
  attempts: 10,
  backoff: 1_000
}

export const CHECK_HEARTBEAT_QUEUE_SETTINGS = {
  removeOnComplete: REMOVE_ON_COMPLETE,
  removeOnFail: REMOVE_ON_FAIL,
  attempts: 10,
  backoff: 1_000,
  repeat: {
    every: 2_000
  }
}

export const LISTENER_JOB_SETTINGS = {
  removeOnComplete: REMOVE_ON_COMPLETE,
  removeOnFail: REMOVE_ON_FAIL,
  attempts: 10,
  backoff: 1_000
}

export const WORKER_JOB_SETTINGS = {
  removeOnComplete: REMOVE_ON_COMPLETE,
  // FIXME Should not be removed until resolved, however, for now in
  // testnet, we can safely keep this settings.
  removeOnFail: REMOVE_ON_FAIL,
  attempts: 10,
  backoff: 1_000
}

export const NONCE_MANAGER_JOB_SETTINGS = {
  removeOnComplete: REMOVE_ON_COMPLETE,
  removeOnFail: REMOVE_ON_FAIL,
  attempts: 10,
  backoff: 500
}

export function getObservedBlockRedisKey(contractAddress: string) {
  return `${contractAddress}-listener-${DEPLOYMENT_NAME}`
}
