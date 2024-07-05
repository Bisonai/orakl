import { describe, expect, test } from '@jest/globals'
import {
  insertHandler as chainInsertHandler,
  listHandler as chainListHandler,
  removeHandler as chainRemoveHandler,
} from '../src/chain'
import { insertHandler, listHandler, removeHandler } from '../src/listener'
import {
  insertHandler as serviceInsertHandler,
  listHandler as serviceListHandler,
  removeHandler as serviceRemoveHandler,
} from '../src/service'

describe('CLI Listener', function () {
  const LISTENER_0 = {
    chain: 'localhost',
    service: 'VRF',
    address: '0x0000000000000000000000000000000000000000',
    eventName: 'Event',
  }

  const LISTENER_1 = {
    chain: 'localhost',
    service: 'VRF',
    address: '0x0000000000000000000000000000000000000001',
    eventName: 'Event',
  }

  let initialListenerId
  beforeAll(async () => {
    await chainInsertHandler()({ name: 'localhost' })
    await serviceInsertHandler()({ name: 'VRF' })
    const insertResult = await insertHandler()(LISTENER_0)
    initialListenerId = insertResult.id
  })

  afterAll(async () => {
    const listeners = await listHandler()({})
    for (const listener of listeners) {
      await removeHandler()({ id: listener.id })
    }
    const chains = await chainListHandler()()
    for (const chain of chains) {
      await chainRemoveHandler()({ id: chain.id })
    }
    const services = await serviceListHandler()()
    for (const service of services) {
      await serviceRemoveHandler()({ id: service.id })
    }
  })

  test('Should list all listeners', async function () {
    const listener = await listHandler()({})
    expect(listener.length).toBeGreaterThan(0)
  })

  test('Should insert new listener', async function () {
    const listenerBefore = await listHandler()({})
    await insertHandler()(LISTENER_1)
    const listenerAfter = await listHandler()({})
    expect(listenerAfter.length).toEqual(listenerBefore.length + 1)
  })

  test('Should delete listener based on id', async function () {
    const listenerBefore = await listHandler()({})
    await removeHandler()({ id: initialListenerId })
    const listenerAfter = await listHandler()({})
    expect(listenerAfter.length).toEqual(listenerBefore.length - 1)
  })
})
