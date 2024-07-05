import { describe, expect, test } from '@jest/globals'
import { insertHandler, listHandler, removeHandler } from '../src/service'

describe('CLI Service', function () {
  let initialServiceId
  beforeAll(async () => {
    const result = await insertHandler()({ name: 'VRF' })
    initialServiceId = result.id
  })
  afterAll(async () => {
    const services = await listHandler()()
    for (const service of services) {
      await removeHandler()({ id: service.id })
    }
  })

  test('Should list service', async function () {
    const service = await listHandler()()
    expect(service.length).toBeGreaterThan(0)
  })

  test('Should insert new service', async function () {
    const serviceBefore = await listHandler()()
    const result = await insertHandler()({ name: 'Automation' })
    const serviceAfter = await listHandler()()
    expect(serviceAfter.length).toEqual(serviceBefore.length + 1)
    await removeHandler()({ id: result.id })
  })

  test('Should not allow to insert the same service more than once', async function () {
    await insertHandler()({ name: 'Automation' })
    const msg = await insertHandler()({ name: 'Automation' })
    expect(msg).toEqual(
      'ERROR: duplicate key value violates unique constraint "services_name_key" (SQLSTATE 23505)',
    )
  })

  test('Should delete service based on id', async function () {
    const serviceBefore = await listHandler()()
    await removeHandler()({ id: initialServiceId })
    const serviceAfter = await listHandler()()
    expect(serviceAfter.length).toEqual(serviceBefore.length - 1)
  })
})
