import { aggregatorMapping } from './aggregator.js'

export const localAggregatorFn =
  aggregatorMapping[process.env.LOCAL_AGGREGATOR?.toUpperCase() || 'MEAN']
