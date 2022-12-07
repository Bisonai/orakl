import { aggregatorMapping } from './aggregator'
import { LOCAL_AGGREGATOR } from './load-parameters'

export const localAggregatorFn = aggregatorMapping[LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']

export const workerRequestQueueName = 'worker-request-queue'

export const reporterRequestQueueName = 'reporter-request-queue'
