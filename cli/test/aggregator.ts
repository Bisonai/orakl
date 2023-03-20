import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/aggregator'

describe('CLI Aggregator', function () {
  const AGGREGATOR = {
    name: 'X-Y',
    address: '0x0000000000000000000000000000000000000000',
    fixedHeartbeatRate: 15_000,
    threshold: 0.05,
    absoluteThreshold: 0.1,
    adapterId: '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5'
  }

  test.skip('Should list Aggregators', async function () {
    const aggregator = await listHandler()({})
    expect(aggregator.length).toBeGreaterThan(0)
  })

  test.skip('Should insert new aggregator', async function () {
    const aggregatorBefore = await listHandler()({})
    await insertHandler()({ data: AGGREGATOR, chain: 'localhost' })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length + 1)
  })

  test.skip('Should not allow to insert the same aggregator more than once', async function () {
    await insertHandler()({ data: AGGREGATOR, chain: 'localhost' })
    await expect(async () => {
      await insertHandler()({ data: AGGREGATOR, chain: 'localhost' })
    }).rejects.toThrow()
  })

  test.skip('Should delete aggregator based on id', async function () {
    const aggregatorBefore = await listHandler()({})
    await removeHandler()({ id: 1 })
    const aggregatorAfter = await listHandler()({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length - 1)
  })
})
