import { Test, TestingModule } from '@nestjs/testing'
import { LastSubmissionService } from './last-submission.service'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { LastSubmissionDto } from './dto/last-submission.dto'
import { AdapterService } from './../adapter/adapter.service'
import { AggregatorService } from './../aggregator/aggregator.service'
import { ChainService } from './../chain/chain.service'

describe('LastSubmissionService', () => {
  let lastsubmision: LastSubmissionService
  let adapter: AdapterService
  let aggregator: AggregatorService
  let chain: ChainService
  let prisma

  let submissionData, submissionObj, adapterObj, aggregatorObj, chainObj

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        LastSubmissionService,
        PrismaService,
        AdapterService,
        AggregatorService,
        ChainService
      ]
    }).compile()
    prisma = module.get<PrismaClient>(PrismaService)
    lastsubmision = module.get<LastSubmissionService>(LastSubmissionService)

    adapter = module.get<AdapterService>(AdapterService)
    chain = module.get<ChainService>(ChainService)
    aggregator = module.get<AggregatorService>(AggregatorService)

    chainObj = await chain.create({ name: 'aggregator-test-chain' })
    const feeds = [
      {
        name: 'Binance-BTC-USD-aggregator',
        definition: {
          url: 'https://api.binance.us/api/v3/ticker/price?symbol=BTCUSD',
          headers: {
            'Content-Type': 'application/json'
          },
          method: 'GET',
          reducers: [
            {
              function: 'PARSE',
              args: ['price']
            },
            {
              function: 'POW10',
              args: 8
            },
            {
              function: 'ROUND'
            }
          ]
        }
      }
    ]
    adapterObj = await adapter.create({
      adapterHash: '0xf9b8b8d3276a46ce36620fe19d98de894b2b493b7ccf77e3800a6aafac6bcfc6',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    // Aggregator
    const aggregatorData = {
      aggregatorHash: '0xc1df01f46e2e128b61a6644aeede3446b2ac86ff0fc7919cadebd0c6938e83b6',
      active: false,
      name: 'BTC-USD',
      address: '0x111',
      heartbeat: 10_000,
      threshold: 0.04,
      absoluteThreshold: 0.1,
      adapterHash: adapterObj.adapterHash,
      chain: chainObj.name
    }

    aggregatorObj = await aggregator.create(aggregatorData)
    submissionData = {
      aggregatorId: BigInt(aggregatorObj.id),
      value: BigInt(1000)
    }

    submissionObj = await lastsubmision.create(submissionData)
  })

  afterEach(async () => {
    // Cleanup
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })

    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(lastsubmision).toBeDefined()
  })

  it('should insert new sumbission record', async () => {
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // The same sumbission cannot be defined twice
    await expect(async () => {
      await lastsubmision.create(submissionData)
    }).rejects.toThrow()

    // Cleanup
    await lastsubmision.remove({ id: submissionObj.id })
  })

  it('should update entity', async () => {
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Update Object
    submissionData.value = BigInt(2000)
    const id = submissionObj.id
    const submissionUpdateObj = await lastsubmision.update({
      where: { id },
      lastSubmissionDto: { ...submissionData }
    })
    expect(submissionUpdateObj.value).toBe(submissionData.value)
    expect(submissionUpdateObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Cleanup
    await lastsubmision.remove({ id: submissionUpdateObj.id })
  })

  it('should upsert', async () => {
    const submissionUpsertObj = await lastsubmision.upsert(submissionData)
    expect(submissionUpsertObj.value).toBe(submissionData.value)
    expect(submissionUpsertObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Cleanup
    await lastsubmision.remove({ id: submissionUpsertObj.id })
  })

  it('should create & upsert entity', async () => {
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Upsert with updated value
    const submissionUpsertData: LastSubmissionDto = {
      aggregatorId: submissionData.aggregatorId,
      value: BigInt(2000)
    }

    const submissionUpsertObj = await lastsubmision.upsert(submissionUpsertData)
    expect(submissionUpsertObj.aggregatorId).toBe(submissionData.aggregatorId)
    expect(submissionUpsertObj.value).not.toEqual(submissionData.value)

    expect(submissionUpsertObj.aggregatorId).toBe(submissionUpsertData.aggregatorId)
    expect(submissionUpsertObj.value).toEqual(submissionUpsertData.value)

    // Cleanup
    await lastsubmision.remove({ id: submissionUpsertObj.id })
  })

  it('should find entity with AggregatorId', async () => {
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Find with Aggregator Id
    const findObj = await lastsubmision.findOne({ aggregatorId: submissionData.aggregatorId })
    expect(findObj.aggregatorId).toBe(submissionData.aggregatorId)
    expect(findObj.value).toBe(submissionData.value)

    // Cleanup
    await lastsubmision.remove({ id: submissionObj.id })
  })

  it('should find entity with AggregatorHash', async () => {
    expect(submissionObj.value).toBe(submissionData.value)
    expect(submissionObj.aggregatorId).toBe(submissionData.aggregatorId)

    // Find with Aggregator Hash
    const findObj = await lastsubmision.findByhash({ aggregatorHash: aggregatorObj.hash })
    expect(findObj.aggregatorId).toBe(submissionData.aggregatorId)
    expect(findObj.value).toBe(submissionData.value)

    // Cleanup
    await lastsubmision.remove({ id: submissionObj.id })
  })
})
