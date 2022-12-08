import { aggregatorMapping } from './aggregator'
import { LOCAL_AGGREGATOR } from './load-parameters'

export const localAggregatorFn = aggregatorMapping[LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']

export const WORKER_REQUEST_QUEUE_NAME = 'worker-request-queue'

export const WORKER_VRF_QUEUE_NAME = 'worker-vrf-queue'

export const REPORTER_REQUEST_QUEUE_NAME = 'reporter-request-queue'

export const REPORTER_VRF_QUEUE_NAME = 'reporter-vrf-queue'
