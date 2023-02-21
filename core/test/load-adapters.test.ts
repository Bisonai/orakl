import { describe, test } from '@jest/globals'
import { fetchDataWithAdapter, loadAdapters } from '../src/worker/utils'

describe('Load All Adapters', function () {
  test('check loadAdapters & fetchDataWithAdapters', async function () {
    const adapters = Object.values(await loadAdapters({ postprocess: false }))
    for (const adapter of adapters) {
      await fetchDataWithAdapter(adapter.feeds)
    }
  })
})
