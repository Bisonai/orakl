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
    sk: '83a8c15d203a71f4f9e7238d663d1ae7eabe10bee47699d4256438acf9bdcce3',
    pk: '044ffbfebcd28f48144c18f7bd9f233199c438b39b5ce1ecc8f049924ba57a8740a814ca7ac5d14c34850e3b61dcbce296de95a4578ac928f8bab48f2a834d1bb9',
    pk_x: '36177951785554001241008675842510466823271112960800516449139368880820117473088',
    pk_y: '76025292965992487548362208012694556435399374398995576443525051210529378212793'
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
