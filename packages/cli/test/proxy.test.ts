import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/proxy'

const proxyData = {
  protocol: 'http',
  host: '127.0.0.1',
  port: 80
}

describe('CLI Proxy', function () {
  test.skip('Should insert new proxy', async function () {
    const proxyBefore = await listHandler()()
    await insertHandler()(proxyData)
    const proxyAfter = await listHandler()()
    expect(proxyAfter.length).toEqual(proxyBefore.length + 1)
  })

  test.skip('Should not allow to insert the same proxy more than once', async function () {
    await insertHandler()(proxyData)
    await expect(async () => {
      await insertHandler()(proxyData)
    }).rejects.toThrow()
  })

  test.skip('Should list proxies', async function () {
    const proxy = await listHandler()()
    expect(proxy.length).toBeGreaterThan(0)
  })

  test.skip('Should delete proxy based on id', async function () {
    const proxyBefore = await listHandler()()
    const lastInstance = proxyBefore[proxyBefore.length - 1]
    await removeHandler()({ id: Number(lastInstance.id) })
    const proxyAfter = await listHandler()()
    expect(proxyAfter.length).toEqual(proxyBefore.length - 1)
  })
})
