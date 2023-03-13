import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/adapter'

describe('CLI Adapter', function () {
  const ADAPTER = {
    active: true,
    name: 'X-Y',
    decimals: '8',
    feeds: [
      {
        name: 'data-X-Y',
        definition: {
          url: 'https://data.com',
          headers: { 'Content-Type': 'application/json' },
          method: 'GET',
          reducers: [
            { function: 'PARSE', args: ['PRICE'] },
            { function: 'POW10', args: '8' },
            { function: 'ROUND' }
          ]
        }
      }
    ]
  }

  test.skip('Should list Adapters', async function () {
    const adapter = await listHandler()()
    expect(adapter.length).toBeGreaterThan(0)
  })

  test.skip('Should insert new adapter', async function () {
    const adapterBefore = await listHandler()()
    await insertHandler()({ data: ADAPTER })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length + 1)
  })

  test.skip('Should not allow to insert the same adapter more than once', async function () {
    await insertHandler()({ data: ADAPTER })
    await expect(async () => {
      await insertHandler()({ data: ADAPTER })
    }).rejects.toThrow()
  })

  test.skip('Should delete adapter based on id', async function () {
    const adapterBefore = await listHandler()()
    await removeHandler()({ id: 1 })
    const adapterAfter = await listHandler()()
    expect(adapterAfter.length).toEqual(adapterBefore.length - 1)
  })
})
