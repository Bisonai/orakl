import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/operator/listener'
import { openDb } from '../src/cli/operator/utils-test'

describe('CLI Listener', function () {
  let DB
  const LISTENER = {
    chain: 'localhost',
    service: 'VRF',
    address: '0x0000000000000000000000000000000000000000',
    eventName: 'Event'
  }
  beforeEach(async () => {
    DB = await openDb({ migrate: true })
  })

  test('Should list all listeners', async function () {
    await listHandler(DB)({})
  })

  test('Should insert new listener', async function () {
    const listenerBefore = await listHandler(DB)({})
    await insertHandler(DB)(LISTENER)
    const listenerAfter = await listHandler(DB)({})
    expect(listenerAfter.length).toEqual(listenerBefore.length + 1)
  })

  test('Should delete listener based on id', async function () {
    const listenerBefore = await listHandler(DB)({})
    await removeHandler(DB)({ id: 1 })
    const listenerAfter = await listHandler(DB)({})
    expect(listenerAfter.length).toEqual(listenerBefore.length - 1)
  })
})
