import * as Path from 'node:path'
import { aggregatorMapping } from './aggregator'
import { LOCAL_AGGREGATOR, REDIS_HOST, REDIS_PORT } from './load-parameters'

export const localAggregatorFn = aggregatorMapping[LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']

export const WORKER_ANY_API_QUEUE_NAME = 'worker-any-API-queue'

export const WORKER_PREDEFINED_FEED_QUEUE_NAME = 'worker-predefined-feed-queue'

export const WORKER_VRF_QUEUE_NAME = 'worker-vrf-queue'

export const REPORTER_ANY_API_QUEUE_NAME = 'reporter-any-api-queue'

export const REPORTER_PREDEFINED_FEED_QUEUE_NAME = 'reporter-predefined-feed-queue'

export const REPORTER_VRF_QUEUE_NAME = 'reporter-vrf-queue'

export const BULLMQ_CONNECTION = {
  connection: {
    host: REDIS_HOST,
    port: REDIS_PORT
  }
}

export const ADAPTER_ROOT_DIR = './adapter/'

export const LISTENER_ROOT_DIR = './tmp/listener/'

export const CONFIG_ROOT_DIR = './config/'

export const LISTENER_CONFIG_FILE = Path.join(CONFIG_ROOT_DIR, 'listener.json')

export const VRF_CONFIG_FILE = Path.join(CONFIG_ROOT_DIR, 'vrf.json')

export const LISTENER_DELAY = 500
