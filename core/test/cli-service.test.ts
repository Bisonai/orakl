import { describe, expect, beforeEach, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/cli/operator/service'
import { openDB } from '../src/cli/operator/utils-test'

describe('CLI Service', function () {
  let DB
  beforeEach(async () => {
    DB = await openDB({ migrate: true })
  })

  test('Should list service', async function () {
    const service = await listHandler(DB)()
    expect(service.length).toBeGreaterThan(0)
  })

  test('Should insert new service', async function () {
    const serviceBefore = await listHandler(DB)()
    await insertHandler(DB)({ name: 'Automation' })
    const serviceAfter = await listHandler(DB)()
    expect(serviceAfter.length).toEqual(serviceBefore.length + 1)
  })

  test('Should not allow to insert the same service more than once', async function () {
    await insertHandler(DB)({ name: 'Automation' })
    await expect(async () => {
      await insertHandler(DB)({ name: 'Automation' })
    }).rejects.toThrow()
  })

  test('Should delete service based on id', async function () {
    const serviceBefore = await listHandler(DB)()
    await removeHandler(DB)({ id: 1 })
    const serviceAfter = await listHandler(DB)()
    expect(serviceAfter.length).toEqual(serviceBefore.length - 1)
  })
})
