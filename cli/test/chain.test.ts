import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/chain'

describe('CLI Chain', function () {
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
    await expect(async () => {
      await insertHandler()({ name: 'ethereum' })
    }).rejects.toThrow()
  })

  test('Should delete chain based on id', async function () {
    const chainBefore = await listHandler()()
    await removeHandler()({ id: 1 })
    const chainAfter = await listHandler()()
    expect(chainAfter.length).toEqual(chainBefore.length - 1)
  })
})
