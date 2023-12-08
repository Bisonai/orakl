import { describe, expect, test } from '@jest/globals'
import {
  insertHandler as adapterInsertHandler,
  listHandler as adapterListHandler,
  removeHandler as adapterRemoveHandler
} from '../src/adapter'
import { insertHandler, listHandler, removeHandler } from '../src/aggregator'
import {
  insertHandler as chainInsertHandler,
  listHandler as chainListHandler,
  removeHandler as chainRemoveHandler
} from '../src/chain'

describe('CLI Aggregator', function () {
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

  const AGGREGATOR_0 = {
    name: 'X-Y',
    aggregatorHash: '0x5bcc6c18d584dc54a666f9212229226f02f65b8dcda3ed72836b6c901f2d18e1',
    address: '0x0000000000000000000000000000000000000000',
    heartbeat: 15000,
    threshold: 0.05,
    absoluteThreshold: 0.1,
    adapterHash: '0x020e150749af3bffaec9ae337da0b9b00c3cfe0b46b854a8e2f5922f6ba2c5db'
  }

  const AGGREGATOR_1 = {
    name: 'Z-X',
    aggregatorHash: '0x11ca65b539221125a64b38653f65dbbf961ed2ea16bcaf54408a5d2ebdc13a0b',
    address: '0x0000000000000000000000000000000000000001',
    heartbeat: 15000,
    threshold: 0.05,
    absoluteThreshold: 0.1,
    adapterHash: '0x12da2f5119ba624ed025303b424d637349c0d120d02bd66a9cfff57e98463a81'
  }

  let initialAggregatorId
  beforeAll(async () => {
    await chainInsertHandler()({ name: 'localhost' })
    await adapterInsertHandler()({ data: ADAPTER_0 })
    await adapterInsertHandler()({ data: ADAPTER_1 })

    const insertResult = await insertHandler()({ data: AGGREGATOR_0, chain: 'localhost' })
    initialAggregatorId = insertResult.id
  })

  afterAll(async () => {
    const aggregators = await listHandler()({})
    for (const aggregator of aggregators) {
      await removeHandler()({ id: aggregator.id })
    }
    const adapters = await adapterListHandler()()
    for (const adapter of adapters) {
      await adapterRemoveHandler()({ id: adapter.id })
    }
    const chains = await chainListHandler()()
    for (const chain of chains) {
      await chainRemoveHandler()({ id: chain.id })
    }
  })

  test('Should list Aggregators', async function () {
    const aggregator = await listHandler()({})
    expect(aggregator.length).toBeGreaterThan(0)
  })

  test('Should insert new aggregator', async function () {
    const aggregatorBefore = await listHandler()({})
    await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length + 1)
  })

  test('Should not allow to insert the same aggregator more than once', async function () {
    await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })

    const msg = await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })
    expect(msg).toEqual('Unique constraint failed on the address')
  })

  test('Should delete aggregator based on id', async function () {
    const aggregatorBefore = await listHandler()({})
    await removeHandler()({ id: initialAggregatorId })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length - 1)
  })
})
