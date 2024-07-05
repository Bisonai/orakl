import { describe, expect, test } from '@jest/globals'
import {
  insertHandler as chainInsertHandler,
  listHandler as chainListHandler,
  removeHandler as chainRemoveHandler,
} from '../src/chain'
import { insertHandler, listHandler, removeHandler } from '../src/vrf'
import { VRF_0, VRF_1 } from './mockData'

describe('CLI Vrf', function () {
  let initialVrfId
  beforeAll(async () => {
    await chainInsertHandler()({ name: 'baobab' })
    await chainInsertHandler()({ name: 'localhost' })
    const insertResult = await insertHandler()(VRF_0)
    initialVrfId = insertResult.id
  })
  afterAll(async () => {
    const vrfs = await listHandler()({})
    for (const vrf of vrfs) {
      await removeHandler()({ id: vrf.id })
    }

    const chains = await chainListHandler()()
    for (const chain of chains) {
      await chainRemoveHandler()({ id: chain.id })
    }
  })

  test('Should list all VRF keys', async function () {
    const vrf = await listHandler()({})
    expect(vrf.length).toBeGreaterThan(0)
  })

  test('Should insert new VRF keys', async function () {
    const vrfBefore = await listHandler()({})
    const insertResult = await insertHandler()(VRF_1)
    const vrfAfter = await listHandler()({})
    expect(vrfAfter.length).toEqual(vrfBefore.length + 1)
    await removeHandler()({ id: insertResult.id })
  })

  test('Should delete VRF based on id', async function () {
    const vrfBefore = await listHandler()({})
    await removeHandler()({ id: initialVrfId })
    const vrfAfter = await listHandler()({})
    expect(vrfAfter.length).toEqual(vrfBefore.length - 1)
  })
})
