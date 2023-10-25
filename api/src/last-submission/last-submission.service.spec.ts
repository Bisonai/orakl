import { Test, TestingModule } from '@nestjs/testing'
import { LastSubmissionService } from './last-submission.service'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { LastSubmissionDto } from './dto/last-submission.dto'

describe('LastSubmissionService', () => {
  let service: LastSubmissionService
  let prisma, submissionData

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [LastSubmissionService, PrismaService]
    }).compile()

    submissionData = {
      aggregatorId: BigInt(1),
      value: BigInt(1000)
    }
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
    const submissionObj = await service.create(submissionData)
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // The same sumbission cannot be defined twice
    await expect(async () => {
      await service.create(submissionData)
    }).rejects.toThrow()

    // Cleanup
    await service.remove({ id: submissionObj.id })
  })

  it('should update entity', async () => {
    const submissionCreateObj = await service.create(submissionData)
    expect(submissionCreateObj.value).toBe(submissionData.value)
    expect(submissionCreateObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Update Object
    submissionData.value = BigInt(2000)
    const id = submissionCreateObj.id
    const submissionUpdateObj = await service.update({
      where: { id },
      lastSubmissionDto: { ...submissionData }
    })
    expect(submissionUpdateObj.value).toBe(submissionData.value)
    expect(submissionUpdateObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Cleanup
    await service.remove({ id: submissionUpdateObj.id })
  })

  it('should upsert', async () => {
    const submissionUpsertObj = await service.upsert(submissionData)
    expect(submissionUpsertObj.value).toBe(submissionData.value)
    expect(submissionUpsertObj.aggregatorId).toBe(submissionData.aggregatorId)
    // Cleanup
    await service.remove({ id: submissionUpsertObj.id })
  })

  it('should create & upsert entity', async () => {
    const submissionCreateObj = await service.create(submissionData)
    expect(submissionCreateObj.value).toBe(submissionData.value)
    expect(submissionCreateObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Upsert with updated value
    const submissionUpsertData: LastSubmissionDto = {
      aggregatorId: submissionData.aggregatorId,
      value: BigInt(2000)
    }

    const submissionUpsertObj = await service.upsert(submissionUpsertData)
    expect(submissionUpsertObj.aggregatorId).toBe(submissionData.aggregatorId)
    expect(submissionUpsertObj.value).not.toEqual(submissionData.value)

    expect(submissionUpsertObj.aggregatorId).toBe(submissionUpsertData.aggregatorId)
    expect(submissionUpsertObj.value).toEqual(submissionUpsertData.value)

    // Cleanup
    await service.remove({ id: submissionUpsertObj.id })
  })

  it('should find entity with AggregatorId', async () => {
    const submissionCreateObj = await service.create(submissionData)

    expect(submissionCreateObj.value).toBe(submissionData.value)
    expect(submissionCreateObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Find with Aggregator Id
    const findObj = await service.findOne({ aggregatorId: submissionData.aggregatorId })
    expect(findObj.aggregatorId).toBe(submissionData.aggregatorId)
    expect(findObj.value).toBe(submissionData.value)
    // Cleanup
    await service.remove({ id: submissionCreateObj.id })
  })
})
