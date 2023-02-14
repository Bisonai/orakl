import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/orakl-cli/chain'
import { openDb } from '../src/cli/orakl-cli/utils'
import { mkTmpFile } from '../src/utils'

describe('CLI Chain', function () {
  let DB
  const TMP_DB_FILE = mkTmpFile({ fileName: 'settings.test.sqlite' })

  beforeEach(async () => {
    DB = await openDb({ dbFile: TMP_DB_FILE, migrate: true })
  })

  test('Should list chain', async function () {
    const chain = await listHandler(DB)()
    expect(chain.length).toBeGreaterThan(0)
  })

  test('Should insert new chain', async function () {
    const chainBefore = await listHandler(DB)()
    await insertHandler(DB)({ name: 'ethereum' })
    const chainAfter = await listHandler(DB)()
    expect(chainAfter.length).toEqual(chainBefore.length + 1)
  })

  test('Should not allow to insert the same chain more than once', async function () {
    await insertHandler(DB)({ name: 'ethereum' })
    await expect(async () => {
      await insertHandler(DB)({ name: 'ethereum' })
    }).rejects.toThrow()
  })

  test('Should delete chain based on id', async function () {
    const chainBefore = await listHandler(DB)()
    await removeHandler(DB)({ id: 1 })
    const chainAfter = await listHandler(DB)()
    expect(chainAfter.length).toEqual(chainBefore.length - 1)
  })
})
