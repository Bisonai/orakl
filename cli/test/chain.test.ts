import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/chain'

describe('CLI Chain', function () {
  let initalChainId
  beforeAll(async () => {
    const insertResult = await insertHandler()({ name: 'boabab' })
    initalChainId = insertResult.id
  })

  afterAll(async () => {
    const chains = await listHandler()()
    for (const chain of chains) {
      await removeHandler()({ id: chain.id })
    }
  })

  test('Should list chain', async function () {
    const chain = await listHandler()()
    expect(chain.length).toBeGreaterThan(0)
  })

  test('Should insert new chain', async function () {
    const chainBefore = await listHandler()()
    await insertHandler()({ name: 'ethereum' })
    const chainAfter = await listHandler()()
    expect(chainAfter.length).toEqual(chainBefore.length + 1)
  })

  test('Should not allow to insert the same chain more than once', async function () {
    await insertHandler()({ name: 'ethereum' })
    const msg = await insertHandler()({ name: 'ethereum' })
    expect(msg).toEqual(
      'ERROR: duplicate key value violates unique constraint "chains_name_key" (SQLSTATE 23505)',
    )
  })

  test('Should delete chain based on id', async function () {
    const chainBefore = await listHandler()()
    await removeHandler()({ id: initalChainId })
    const chainAfter = await listHandler()()
    expect(chainAfter.length).toEqual(chainBefore.length - 1)
  })
})
