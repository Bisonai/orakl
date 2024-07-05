import { mean, median } from 'mathjs'

const AGGREGATOR_MAPPING = {
  MEAN: (i) => Math.round(mean(i)),
  MEDIAN: (i) => Math.round(median(i)),
}

const LOCAL_AGGREGATOR = process.env.LOCAL_AGGREGATOR?.toUpperCase() || 'MEDIAN'
export const LOCAL_AGGREGATOR_FN = AGGREGATOR_MAPPING[LOCAL_AGGREGATOR]
