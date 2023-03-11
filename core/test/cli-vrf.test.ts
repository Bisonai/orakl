import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/orakl-cli/src/vrf'
import { openDb } from '../src/cli/orakl-cli/src/utils'
import { mkTmpFile } from '../src/utils'
import { MIGRATIONS_PATH } from '../src/settings'

describe('CLI Vrf', function () {
  let DB
  const TMP_DB_FILE = mkTmpFile({ fileName: 'settings.test.sqlite' })
  const VRF = {
    chain: 'baobab',
    sk: 'adcfaf9a860722a89472884a2aab4a62f06a42fd4bee55f2fc7f2f11b07f1d81',
    pk: '041f058731839e8c2fb3a77a4be788520f1743f1298a84bd138871f31ffdee04e42b4f962995ba0135eed67f3ebd1739d4b09f1b84224c0d6765e5f426b25443a4',
    pk_x: '14031465612060486287063884409830887522455901523026705297854775800516553082084',
    pk_y: '19590069790275828365845547074408283587257770205538752975574862882950389973924',
    key_hash: '0x956506aeada5568c80c984b908e9e1af01bd96709977b0b5cb1957736e80e883'
  }

  beforeEach(async () => {
    DB = await openDb({ dbFile: TMP_DB_FILE, migrate: true, migrationsPath: MIGRATIONS_PATH })
  })

  test('Should list all VRF keys', async function () {
    const vrf = await listHandler(DB)({})
    expect(vrf.length).toBeGreaterThan(0)
  })

  test('Should insert new VRF keys', async function () {
    const vrfBefore = await listHandler(DB)({})
    await insertHandler(DB)(VRF)
    const vrfAfter = await listHandler(DB)({})
    expect(vrfAfter.length).toEqual(vrfBefore.length + 1)
  })

  test('Should delete VRF based on id', async function () {
    const vrfBefore = await listHandler(DB)({})
    await removeHandler(DB)({ id: 1 })
    const vrfAfter = await listHandler(DB)({})
    expect(vrfAfter.length).toEqual(vrfBefore.length - 1)
  })
})
