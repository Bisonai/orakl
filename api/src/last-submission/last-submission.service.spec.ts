import { Test, TestingModule } from '@nestjs/testing'
import { LastSubmissionService } from './last-submission.service'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'

describe('LastSubmissionService', () => {
  let service: LastSubmissionService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [LastSubmissionService]
    }).compile()

    service = module.get<LastSubmissionService>(LastSubmissionService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })

  it('should insert new sumbission record', async () => {
    const submissionData = {
      aggregatorId: 1,
      timestamp: new Date(),
      value: 123
    }
    const proxyObj = await service.create(submissionData)
    expect(proxyObj.protocol).toBe(proxyData.protocol)
    expect(proxyObj.host).toBe(proxyData.host)
    expect(proxyObj.port).toBe(proxyData.port)

    // The same proxy cannot be defined twice
    await expect(async () => {
      await proxy.create(proxyData)
    }).rejects.toThrow()

    // Cleanup
    await proxy.remove({ id: proxyObj.id })
  })
})
