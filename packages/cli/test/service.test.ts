import { describe, expect, test } from '@jest/globals'
import { listHandler, insertHandler, removeHandler } from '../src/service'

describe('CLI Service', function () {
  test.skip('Should list service', async function () {
    const service = await listHandler()()
    expect(service.length).toBeGreaterThan(0)
  })

  test.skip('Should insert new service', async function () {
    const serviceBefore = await listHandler()()
    await insertHandler()({ name: 'Automation' })
    const serviceAfter = await listHandler()()
    expect(serviceAfter.length).toEqual(serviceBefore.length + 1)
  })

  test.skip('Should not allow to insert the same service more than once', async function () {
    await insertHandler()({ name: 'Automation' })
    await expect(async () => {
      await insertHandler()({ name: 'Automation' })
    }).rejects.toThrow()
  })

  test.skip('Should delete service based on id', async function () {
    const serviceBefore = await listHandler()()
    await removeHandler()({ id: 1 })
    const serviceAfter = await listHandler()()
    expect(serviceAfter.length).toEqual(serviceBefore.length - 1)
  })
})
