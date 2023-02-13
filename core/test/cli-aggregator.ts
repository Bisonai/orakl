import { describe, expect, beforeEach, test } from '@jest/globals'
import {
  listHandler,
  insertHandler,
  removeHandler,
  insertFromChainHandler
} from '../src/cli/operator/aggregator'
import { mkTmpFile } from '../src/utils'
import { openDb } from '../src/cli/operator/utils'

describe('CLI Aggregator', function () {
  let DB
  const TMP_DB_FILE = mkTmpFile({ fileName: 'settings.test.sqlite' })
  const ADAPTER_ID = '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5'
  const AGGREGATOR = {
    address: '0x0000000000000000000000000000000000000000',
    active: true,
    name: 'X/Y',
    fixedHeartbeatRate: { active: true, value: 15000 },
    randomHeartbeatRate: { active: true, value: 2000 },
    threshold: 0.05,
    absoluteThreshold: 0.1,
    adapterId: '0x00d5130063bee77302b133b5c6a0d6aede467a599d251aec842d24abeb5866a5'
  }

  beforeEach(async () => {
    DB = await openDb({ dbFile: TMP_DB_FILE, migrate: true })
  })

  test('Should list Aggregators', async function () {
    const aggregator = await listHandler(DB)({})
    expect(aggregator.length).toBeGreaterThan(0)
  })

  test('Should insert new aggregator', async function () {
    const aggregatorBefore = await listHandler(DB)({})
    await insertHandler(DB)({ data: AGGREGATOR, adapter: ADAPTER_ID, chain: 'localhost' })
    const aggregatorAfter = await listHandler(DB)({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length + 1)
  })

  test('Should not allow to insert the same aggregator more than once', async function () {
    await insertHandler(DB)({ data: AGGREGATOR, adapter: ADAPTER_ID, chain: 'localhost' })
    await expect(async () => {
      await insertHandler(DB)({ data: AGGREGATOR, adapter: ADAPTER_ID, chain: 'localhost' })
    }).rejects.toThrow()
  })

  test('Should delete aggregator based on id', async function () {
    const aggregatorBefore = await listHandler(DB)({})
    await removeHandler(DB)({ id: 1 })
    const aggregatorAfter = await listHandler(DB)({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length - 1)
  })
  test('Should insert new aggregator from other chain', async function () {
    const firstChain = 'localhost'
    await insertHandler(DB)({ data: AGGREGATOR, adapter: ADAPTER_ID, chain: firstChain })
    const aggregatorBefore = await listHandler(DB)({})
    const aggregatorId = JSON.parse(aggregatorBefore[aggregatorBefore.length - 1].data).id
    await insertFromChainHandler(DB)({
      aggregatorId,
      adapter: ADAPTER_ID,
      fromChain: firstChain,
      toChain: 'baobab'
    })
    const aggregatorAfter = await listHandler(DB)({})
    expect(aggregatorAfter.length).toEqual(aggregatorBefore.length + 1)
  })
})
