import { describe, expect, beforeEach, test } from '@jest/globals'
import {
  listHandler,
  insertHandler,
  removeHandler,
  insertFromChainHandler
} from '../src/cli/orakl-cli/adapter'
import { openDb } from '../src/cli/orakl-cli/utils'
import { mkTmpFile } from '../src/utils'

describe('CLI Adapter', function () {
  let DB
  const TMP_DB_FILE = mkTmpFile({ fileName: 'settings.test.sqlite' })
  const ADAPTER = {
    active: true,
    name: 'X/Y',
    jobType: 'JOB',
    decimals: '8',
    feeds: [
      {
        url: 'https://data.com',
        headers: { 'Content-Type': 'application/json' },
        method: 'GET',
        reducers: [
          { function: 'PARSE', args: ['PRICE'] },
          { function: 'POW10', args: '8' },
          { function: 'ROUND' }
        ]
      }
    ]
  }

  beforeEach(async () => {
    DB = await openDb({ dbFile: TMP_DB_FILE, migrate: true })
  })

  test('Should list Adapters', async function () {
    const adapter = await listHandler(DB)({})
    expect(adapter.length).toBeGreaterThan(0)
  })

  test('Should insert new adapter', async function () {
    const adapterBefore = await listHandler(DB)({})
    await insertHandler(DB)({ data: ADAPTER, chain: 'localhost' })
    const adapterAfter = await listHandler(DB)({})
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
  })

  test('Should not allow to insert the same adapter more than once', async function () {
    await insertHandler(DB)({ data: ADAPTER, chain: 'localhost' })
    await expect(async () => {
      await insertHandler(DB)({ data: ADAPTER, chain: 'localhost' })
    }).rejects.toThrow()
  })

  test('Should delete adapter based on id', async function () {
    const adapterBefore = await listHandler(DB)({})
    await removeHandler(DB)({ id: 1 })
    const adapterAfter = await listHandler(DB)({})
    expect(adapterAfter.length).toEqual(adapterBefore.length - 1)
  })

  test('Should insert new adapter from other chain', async function () {
    const firstChain = 'localhost'
    await insertHandler(DB)({ data: ADAPTER, chain: firstChain })
    const adapterBefore = await listHandler(DB)({})
    const adapterId = JSON.parse(adapterBefore[adapterBefore.length - 1].data).id
    await insertFromChainHandler(DB)({
      adapterId,
      fromChain: firstChain,
      toChain: 'baobab'
    })
    const adapterAfter = await listHandler(DB)({})
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
  })
})
