import { describe, expect, beforeEach, test } from '@jest/globals'
import {
  listHandler,
  insertHandler,
  insertManyHandler,
  removeHandler,
  updateHandler
} from '../src/cli/operator/kv'
import { mkTmpFile } from '../src/utils'
import { openDb } from '../src/cli/operator/utils'

describe('CLI KV', function () {
  let DB
  const TMP_DB_FILE = mkTmpFile({ fileName: 'settings.test.sqlite' })
  const KV_LOCALHOST = { key: 'someKey', value: 'someValue', chain: 'localhost' }
  const KV_BAOBAB = { key: 'someKey', value: 'someValue', chain: 'baobab' }
  const KV_MANY_LOCALHOST = {
    chain: 'localhost',
    data: [{ key1: 'val1' }, { key2: 'val2' }]
  }

  beforeEach(async () => {
    DB = await openDb({ dbFile: TMP_DB_FILE, migrate: true })
  })

  test('Should list all Key-Value pairs', async function () {
    const kv = await listHandler(DB)({})
    expect(kv.length).toBeGreaterThan(0)
  })

  test('Should list all Key-Value pairs for localhost', async function () {
    const kv = await listHandler(DB)({ chain: 'localhost' })
    expect(kv.length).toBeGreaterThan(0)
  })

  test('Should list all PRIVATE_KEY keys in all chains', async function () {
    const kv = await listHandler(DB)({ key: 'PRIVATE_KEY' })
    expect(kv.length).toBeGreaterThan(0)
  })

  test('Should list single PRIVATE_KEY key for localhost chain', async function () {
    const kv = await listHandler(DB)({ key: 'PRIVATE_KEY', chain: 'localhost' })
    expect(kv.length).toEqual(1)
  })

  test('Should insert new Key-Value pair', async function () {
    const kvBefore = await listHandler(DB)({})
    await insertHandler(DB)(KV_LOCALHOST)
    const kvAfter = await listHandler(DB)({})
    expect(kvAfter.length).toEqual(kvBefore.length + 1)
  })

  test('Should insertMany new Key-Value pairs', async function () {
    const kvBefore = await listHandler(DB)({})
    await insertManyHandler(DB)(KV_MANY_LOCALHOST)
    const kvAfter = await listHandler(DB)({})
    expect(kvAfter.length).toEqual(kvBefore.length + KV_MANY_LOCALHOST.data.length)
  })

  test('Should not allow to insert the same Key-Value pair more than once in the same chain', async function () {
    await insertHandler(DB)(KV_LOCALHOST)
    await expect(async () => {
      await insertHandler(DB)(KV_LOCALHOST)
    }).rejects.toThrow()
  })

  test('Should allow to insert the same Key-Value pair in different chains', async function () {
    await insertHandler(DB)(KV_LOCALHOST)
    await insertHandler(DB)(KV_BAOBAB)
  })

  test('Should delete Key-Value pair specified by key and chain', async function () {
    await insertHandler(DB)(KV_LOCALHOST)
    const kvBefore = await listHandler(DB)({})
    await removeHandler(DB)({ key: KV_LOCALHOST.key, chain: KV_LOCALHOST.chain })
    const kvAfter = await listHandler(DB)({})
    expect(kvAfter.length).toEqual(kvBefore.length - 1)
  })

  test('Should update value of already inserted Key-Value pair specified by key and chain', async function () {
    await insertHandler(DB)(KV_LOCALHOST)
    const newValue = 'newValue'
    await updateHandler(DB)({ key: 'someKey', value: newValue, chain: 'localhost' })
    const kv = await listHandler(DB)({ key: 'someKey', chain: 'localhost' })
    expect(kv.length).toEqual(1)
    expect(kv[0].value).toEqual(newValue)
  })
})
