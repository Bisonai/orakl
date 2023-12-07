import { describe, expect, test } from '@jest/globals'
import { hashHandler, insertHandler, listHandler, removeHandler } from '../src/adapter'

describe('CLI Adapter', function () {
  const ADAPTER = {
    active: true,
    name: 'X-Y',
    decimals: 8,
    feeds: [
      {
        name: 'data-X-Y',
        definition: {
          url: 'https://data.com',
          headers: { 'Content-Type': 'application/json' },
          method: 'GET',
          reducers: [
            { function: 'PARSE', args: ['PRICE'] },
            { function: 'POW10', args: '8' },
            { function: 'ROUND' }
          ]
        }
      }
    ]
  }

  let initalAdapterId
  beforeAll(async () => {
    // setup hash
    const initAdapter = { ...ADAPTER, name: 'Z-X' }
    initAdapter['adapterHash'] = (
      await hashHandler()({ data: initAdapter, verify: false })
    ).adapterHash
    ADAPTER['adapterHash'] = (await hashHandler()({ data: ADAPTER, verify: false })).adapterHash

    // insert default adapter
    const insertResult = await insertHandler()({ data: initAdapter })
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
    await insertHandler()({ data: ADAPTER })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
  })

  test('Should not allow to insert the same adapter more than once', async function () {
    await insertHandler()({ data: ADAPTER })
    const msg = await insertHandler()({ data: ADAPTER })
    expect(msg).toEqual('Unique constraint failed on the adapter_hash')
  })

  test('Should delete adapter based on id', async function () {
    const adapterBefore = await listHandler()()
    await removeHandler()({ id: initalAdapterId })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length - 1)
  })
})
