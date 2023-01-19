import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/operator/chain'
import { openDb } from '../src/cli/operator/utils-test'

describe('CLI Chain', function () {
  let db
  beforeEach(async () => {
    db = await openDb({ migrate: true })
  })

  test('Should list chain', async function () {
    await listHandler(db)()
  })

  test('Should insert new chain', async function () {
    const chainBefore = await listHandler(db)()
    await insertHandler(db)({ name: 'ethereum' })
    const chainAfter = await listHandler(db)()
    expect(chainAfter.length).toEqual(chainBefore.length + 1)
  })

  test('Should not allow to insert the same chain more than once', async function () {
    await insertHandler(db)({ name: 'ethereum' })
    await expect(async () => {
      await insertHandler(db)({ name: 'ethereum' })
    }).rejects.toThrow()
  })

  test('Should delete chain based on id', async function () {
    const chainBefore = await listHandler(db)()
    await removeHandler(db)({ id: 1 })
    const chainAfter = await listHandler(db)()
    expect(chainAfter.length).toEqual(chainBefore.length - 1)
  })
})
