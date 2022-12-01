import { aggregatorMapping } from './aggregator.js'

export const localAggregatorFn =
  aggregatorMapping[process.env.LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']

export const workerRequestQueueName = 'worker-request-queue'

export const reporterRequestQueueName = 'reporter-request-queue'
