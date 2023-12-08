import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/adapter'

describe('CLI Adapter', function () {
  const ADAPTER_0 = {
    active: true,
    name: 'X-Y',
    decimals: 8,
    adapterHash: '0x020e150749af3bffaec9ae337da0b9b00c3cfe0b46b854a8e2f5922f6ba2c5db',
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

  const ADAPTER_1 = {
    active: true,
    name: 'Z-X',
    decimals: 8,
    adapterHash: '0x12da2f5119ba624ed025303b424d637349c0d120d02bd66a9cfff57e98463a81',
    feeds: [
      {
        name: 'data-Z-X',
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
    await insertHandler()({ data: ADAPTER_1 })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
  })

  test('Should not allow to insert the same adapter more than once', async function () {
    await insertHandler()({ data: ADAPTER_1 })
    const msg = await insertHandler()({ data: ADAPTER_1 })
    expect(msg).toEqual('Unique constraint failed on the adapter_hash')
  })

  test('Should delete adapter based on id', async function () {
    const adapterBefore = await listHandler()()
    await removeHandler()({ id: initalAdapterId })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length - 1)
  })
})
