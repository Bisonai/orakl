import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/listener'

describe('CLI Listener', function () {
  const LISTENER = {
    chain: 'localhost',
    service: 'VRF',
    address: '0x0000000000000000000000000000000000000000',
    eventName: 'Event'
  }

  test.skip('Should list all listeners', async function () {
    const listener = await listHandler()({})
    expect(listener.length).toBeGreaterThan(0)
  })

  test.skip('Should insert new listener', async function () {
    const listenerBefore = await listHandler()({})
    await insertHandler()(LISTENER)
    const listenerAfter = await listHandler()({})
    expect(listenerAfter.length).toEqual(listenerBefore.length + 1)
  })

  test.skip('Should delete listener based on id', async function () {
    const listenerBefore = await listHandler()({})
    await removeHandler()({ id: 1 })
    const listenerAfter = await listHandler()({})
    expect(listenerAfter.length).toEqual(listenerBefore.length - 1)
  })
})
