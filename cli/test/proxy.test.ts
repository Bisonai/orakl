import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/proxy'

describe('CLI Proxy', function () {
  const proxyData_0 = {
    protocol: 'http',
    host: '127.0.0.1',
    port: 80,
  }

  const proxyData_1 = {
    protocol: 'http',
    host: '127.0.0.2',
    port: 80,
  }

  let initialProxyId
  beforeAll(async () => {
    const insertResult = await insertHandler()(proxyData_0)
    initialProxyId = insertResult.id
  })
  afterAll(async () => {
    const proxies = await listHandler()()
    for (const proxy of proxies) {
      await removeHandler()({ id: proxy.id })
    }
  })

  test('Should list proxies', async function () {
    const proxy = await listHandler()()
    expect(proxy.length).toBeGreaterThan(0)
  })

  test('Should insert new proxy', async function () {
    const proxyBefore = await listHandler()()
    const result = await insertHandler()(proxyData_1)
    const proxyAfter = await listHandler()()
    expect(proxyAfter.length).toEqual(proxyBefore.length + 1)
    await removeHandler()({ id: result.id })
  })

  test('Should not allow to insert the same proxy more than once', async function () {
    await insertHandler()(proxyData_1)
    const msg = await insertHandler()(proxyData_1)
    expect(msg).toEqual(
      'ERROR: duplicate key value violates unique constraint "proxies_protocol_host_port_key" (SQLSTATE 23505)',
    )
  })

  test('Should delete proxy based on id', async function () {
    const proxyBefore = await listHandler()()
    await removeHandler()({ id: initialProxyId })
    const proxyAfter = await listHandler()()
    expect(proxyAfter.length).toEqual(proxyBefore.length - 1)
  })
})
