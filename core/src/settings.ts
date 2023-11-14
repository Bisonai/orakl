import * as dotenv from 'dotenv'
import { ethers } from 'ethers'
dotenv.config()

export const ORAKL_NETWORK_API_URL =
  process.env.ORAKL_NETWORK_API_URL || 'http://localhost:3000/api/v1'
export const ORAKL_NETWORK_DELEGATOR_URL =
  process.env.ORAKL_NETWORK_DELEGATOR_URL || 'http://localhost:3002/api/v1'

export const DEPLOYMENT_NAME = process.env.DEPLOYMENT_NAME || 'orakl'
export const NODE_ENV = process.env.NODE_ENV
export const HEALTH_CHECK_PORT = process.env.HEALTH_CHECK_PORT
export const CHAIN = process.env.CHAIN || 'localhost'
export const LOG_LEVEL = process.env.LOG_LEVEL || 'info'
export const STORE_ADAPTER_FETCH_RESULT = process.env.STORE_ADAPTER_FETCH_RESULT || false

export const PROVIDER_URL = process.env.PROVIDER_URL || 'http://127.0.0.1:8545'
export const REDIS_HOST = process.env.REDIS_HOST || 'localhost'
export const REDIS_PORT = process.env.REDIS_PORT ? Number(process.env.REDIS_PORT) : 6379
export const SLACK_WEBHOOK_URL = process.env.SLACK_WEBHOOK_URL || ''
export const LOCAL_AGGREGATOR = process.env.LOCAL_AGGREGATOR || 'MEDIAN'
export const LISTENER_DELAY = Number(process.env.LISTENER_DELAY) || 500

// Gas mimimums
export const VRF_FULFILL_GAS_MINIMUM = 1_000_000
export const REQUEST_RESPONSE_FULFILL_GAS_MINIMUM = 400_000
export const DATA_FEED_FULFILL_GAS_MINIMUM = 400_000
export const VRF_FULLFILL_GAS_PER_WORD = 1_000

// Service ports are used for communication to watchman from the outside
export const LISTENER_PORT = process.env.LISTENER_PORT || 4_000
export const WORKER_PORT = process.env.WORKER_PORT || 5_001
export const REPORTER_PORT = process.env.REPORTER_PORT || 6_000

export const DATA_FEED_SERVICE_NAME = 'DATA_FEED'
export const VRF_SERVICE_NAME = 'VRF'
export const REQUEST_RESPONSE_SERVICE_NAME = 'REQUEST_RESPONSE'
export const L2_DATA_FEED_SERVICE_NAME = 'DATA_FEED_L2'

// Data Feed
export const MAX_DATA_STALENESS = 5_000

// BullMQ
export const REMOVE_ON_COMPLETE = 500
export const REMOVE_ON_FAIL = 1_000
export const CONCURRENCY = 12
export const DATA_FEED_REPORTER_CONCURRENCY =
  Number(process.env.DATA_FEED_REPORTER_CONCURRENCY) || 15

export const LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-latest-queue`
export const LISTENER_VRF_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-latest-queue`
export const LISTENER_DATA_FEED_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-latest-queue`
export const L2_LISTENER_DATA_FEED_LATEST_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-latest-queue`

export const LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-history-queue`
export const LISTENER_VRF_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-history-queue`
export const LISTENER_DATA_FEED_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-history-queue`
export const L2_LISTENER_DATA_FEED_HISTORY_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-history-queue`

export const LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-request-response-process-event-queue`
export const LISTENER_VRF_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-vrf-process-event-queue`
export const LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-process-event-queue`
export const L2_LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME = `${DEPLOYMENT_NAME}-listener-data-feed-l2-process-event-queue`

export const SUBMIT_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-submitheartbeat-queue`
export const HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-heartbeat-queue`
export const WORKER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-queue`
export const WORKER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-queue`
export const WORKER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-aggregator-queue`
export const WORKER_CHECK_HEARTBEAT_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-checkheartbeat-queue`
export const L2_WORKER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-worker-aggregator-l2-queue`

export const REPORTER_REQUEST_RESPONSE_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-queue`
export const REPORTER_VRF_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-queue`
export const REPORTER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-aggregator-queue`
export const WORKER_DEVIATION_QUEUE_NAME = `orakl-deviation-queue`
export const L2_REPORTER_AGGREGATOR_QUEUE_NAME = `${DEPLOYMENT_NAME}-reporter-aggregator-l2-queue`

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

// export const VRF_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-vrf-state`
// export const REQUEST_RESPONSE_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-request-response-state`
export const DATA_FEED_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-data-feed-state`
export const L2_DATA_FEED_WORKER_STATE_NAME = `${DEPLOYMENT_NAME}-worker-data-feed-l2-state`

export const VRF_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-vrf-state`
export const REQUEST_RESPONSE_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-request-response-state`
export const DATA_FEED_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-data-feed-state`
export const L2_DATA_FEED_REPORTER_STATE_NAME = `${DEPLOYMENT_NAME}-reporter-data-feed-l2-state`

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

export function getObservedBlockRedisKey(contractAddress: string) {
  return `${contractAddress}-listener-${DEPLOYMENT_NAME}`
}
