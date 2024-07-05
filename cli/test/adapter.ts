import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/adapter'
import { ADAPTER_0, ADAPTER_1 } from './mockData'

describe('CLI Adapter', function () {
  let initalAdapterId
  beforeAll(async () => {
    // insert default adapter
    const insertResult = await insertHandler()({ data: ADAPTER_0 })
    initalAdapterId = insertResult.id
  })

  afterAll(async () => {
    const adapters = await listHandler()()
    for (const adapter of adapters) {
      await removeHandler()({ id: adapter.id })
    }
  })

  test('Should list Adapters', async function () {
    const adapter = await listHandler()()
    expect(adapter.length).toBeGreaterThan(0)
  })

  test('Should insert new adapter', async function () {
    const adapterBefore = await listHandler()()
    const result = await insertHandler()({ data: ADAPTER_1 })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
    await removeHandler()({ id: result.id })
  })

  test('Should not allow to insert the same adapter more than once', async function () {
    await insertHandler()({ data: ADAPTER_1 })
    const msg = await insertHandler()({ data: ADAPTER_1 })
    expect(msg).toEqual(
      'ERROR: duplicate key value violates unique constraint "adapters_adapter_hash_key" (SQLSTATE 23505)',
    )
  })

  test('Should delete adapter based on id', async function () {
    const adapterBefore = await listHandler()()
    await removeHandler()({ id: initalAdapterId })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length - 1)
  })
})
