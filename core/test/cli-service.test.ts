import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/operator/service'
import { openDb } from '../src/cli/operator/utils-test'

describe('CLI Service', function () {
  let db
  beforeEach(async () => {
    db = await openDb({ migrate: true })
  })

  test('Should list service', async function () {
    await listHandler(db)()
  })

  test('Should insert new service', async function () {
    const serviceBefore = await listHandler(db)()
    await insertHandler(db)({ name: 'Automation' })
    const serviceAfter = await listHandler(db)()
    expect(serviceAfter.length).toEqual(serviceBefore.length + 1)
  })

  test('Should not allow to insert the same service more than once', async function () {
    await insertHandler(db)({ name: 'Automation' })
    await expect(async () => {
      await insertHandler(db)({ name: 'Automation' })
    }).rejects.toThrow()
  })

  test('Should delete service based on id', async function () {
    const serviceBefore = await listHandler(db)()
    await removeHandler(db)({ id: 1 })
    const serviceAfter = await listHandler(db)()
    expect(serviceAfter.length).toEqual(serviceBefore.length - 1)
  })
})
