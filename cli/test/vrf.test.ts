import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/vrf'

describe('CLI Vrf', function () {
  const VRF = {
    chain: 'baobab',
    sk: 'adcfaf9a860722a89472884a2aab4a62f06a42fd4bee55f2fc7f2f11b07f1d81',
    pk: '041f058731839e8c2fb3a77a4be788520f1743f1298a84bd138871f31ffdee04e42b4f962995ba0135eed67f3ebd1739d4b09f1b84224c0d6765e5f426b25443a4',
    pkX: '14031465612060486287063884409830887522455901523026705297854775800516553082084',
    pkY: '19590069790275828365845547074408283587257770205538752975574862882950389973924',
    keyHash: '0x956506aeada5568c80c984b908e9e1af01bd96709977b0b5cb1957736e80e883'
  }

  test.skip('Should list all VRF keys', async function () {
    const vrf = await listHandler()({})
    expect(vrf.length).toBeGreaterThan(0)
  })

  test.skip('Should insert new VRF keys', async function () {
    const vrfBefore = await listHandler()({})
    await insertHandler()(VRF)
    const vrfAfter = await listHandler()({})
    expect(vrfAfter.length).toEqual(vrfBefore.length + 1)
  })

  test.skip('Should delete VRF based on id', async function () {
    const vrfBefore = await listHandler()({})
    await removeHandler()({ id: 1 })
    const vrfAfter = await listHandler()({})
    expect(vrfAfter.length).toEqual(vrfBefore.length - 1)
  })
})
