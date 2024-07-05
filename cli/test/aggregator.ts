import { describe, expect, test } from '@jest/globals'
import {
  insertHandler as adapterInsertHandler,
  listHandler as adapterListHandler,
  removeHandler as adapterRemoveHandler,
} from '../src/adapter'
import { insertHandler, listHandler, removeHandler } from '../src/aggregator'
import {
  insertHandler as chainInsertHandler,
  listHandler as chainListHandler,
  removeHandler as chainRemoveHandler,
} from '../src/chain'
import { ADAPTER_0, ADAPTER_1, AGGREGATOR_0, AGGREGATOR_1 } from './mockData'

describe('CLI Aggregator', function () {
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
    const result = await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length + 1)
    await removeHandler()({ id: result.id })
  })

  test('Should not allow to insert the same aggregator more than once', async function () {
    await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })

    const msg = await insertHandler()({ data: AGGREGATOR_1, chain: 'localhost' })
    expect(msg).toEqual(
      'ERROR: duplicate key value violates unique constraint "aggregators_address_key" (SQLSTATE 23505)',
    )
  })

  test('Should delete aggregator based on id', async function () {
    const aggregatorBefore = await listHandler()({})
    await removeHandler()({ id: initialAggregatorId })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length - 1)
  })
})
