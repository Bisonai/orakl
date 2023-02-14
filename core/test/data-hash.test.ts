import { describe, test } from '@jest/globals'
import { loadAdapters, loadAggregators } from '../src/worker/utils'
import { computeDataHash } from '../src/cli/orakl-cli/utils'

describe('Data Hash Checks', function () {
  test('Adapters Hash Check', async function () {
    const adapters = await loadAdapters({ postprocess: false })
    for (const a of adapters) {
      await computeDataHash({ data: a, verify: true })
    }
  })

  test('Aggregator Hash Check', async function () {
    const aggregators = await loadAggregators({ postprocess: false })
    for (const a of aggregators) {
      await computeDataHash({ data: a, verify: true })
    }
  })
})
