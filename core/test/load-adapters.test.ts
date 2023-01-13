import { describe, expect, test } from '@jest/globals'
import { fetchDataWithAdapter, loadAdapters } from '../src/worker/utils'

describe('Load All Adapters', function () {
  test('check loadAdapters & fetchDataWithAdapters', async function () {
    const adapters = await loadAdapters()
    for (const i in adapters) {
      await fetchDataWithAdapter(adapters[i])
    }
  })
})
