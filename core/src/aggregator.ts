import { mean, median } from 'mathjs'

export const aggregatorMapping = {
  MEAN: (i) => Math.round(mean(i)),
  MEDIAN: (i) => Math.round(median(i))
}
