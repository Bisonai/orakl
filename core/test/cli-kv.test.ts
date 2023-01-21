import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/operator/kv'
import { openDb } from '../src/cli/operator/utils-test'

describe('CLI KV', function () {
  let DB
  let KV_LOCALHOST = { key: 'someKey', value: 'someValue', chain: 'localhost' }
  let KV_BAOBAB = { key: 'someKey', value: 'someValue', chain: 'baobab' }
  beforeEach(async () => {
    DB = await openDb({ migrate: true })
  })

  test('Should list all Key-Value pairs', async function () {
    const kv = await listHandler(DB)({})
    expect(kv.length).toBeGreaterThan(0)
  })

  test('Should list all Key-Value pairs for localhost', async function () {
    const kv = await listHandler(DB)({ chain: 'localhost' })
    expect(kv.length).toBeGreaterThan(0)
  })

  test('Should insert new Key-Value pair', async function () {
    const kvBefore = await listHandler(DB)({})
    await insertHandler(DB)(KV_LOCALHOST)
    const kvAfter = await listHandler(DB)({})
    expect(kvAfter.length).toEqual(kvBefore.length + 1)
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
})
