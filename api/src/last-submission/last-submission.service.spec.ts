import { Test, TestingModule } from '@nestjs/testing'
import { LastSubmissionService } from './last-submission.service'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AdapterService } from './../adapter/adapter.service'
import { AggregatorService } from './../aggregator/aggregator.service'
import { ChainService } from './../chain/chain.service'

describe('LastSubmissionService', () => {
  let lastSubmission: LastSubmissionService
  let adapter: AdapterService
  let aggregator: AggregatorService
  let chain: ChainService
  let prisma
  let submissionData, adapterObj, aggregatorObj, chainObj

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
    lastSubmission = module.get<LastSubmissionService>(LastSubmissionService)

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
      aggregatorId: Number(aggregatorObj.id),
      value: Number(1000)
    }
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
    expect(lastSubmission).toBeDefined()
  })

  it('should upsert', async () => {
    const submissionUpsertObj = await lastSubmission.upsert(submissionData)
    expect(submissionUpsertObj.value).toBe(BigInt(submissionData.value))
    expect(submissionUpsertObj.aggregatorId).toBe(BigInt(submissionData.aggregatorId))

    // Cleanup
    await prisma.lastSubmission.delete({ where: { id: submissionUpsertObj.id } })
  })

  it('should find entity with AggregatorHash', async () => {
    const submissionUpsertObj = await lastSubmission.upsert(submissionData)
    expect(submissionUpsertObj.value).toBe(BigInt(submissionData.value))
    expect(submissionUpsertObj.aggregatorId).toBe(BigInt(submissionData.aggregatorId))

    // Find with Aggregator Hash
    const findObj = await lastSubmission.findByhash({ aggregatorHash: aggregatorObj.hash })
    expect(findObj.aggregatorId).toBe(BigInt(submissionUpsertObj.aggregatorId))
    expect(findObj.value).toBe(submissionUpsertObj.value)

    // Cleanup
    await prisma.lastSubmission.delete({ where: { id: submissionUpsertObj.id } })
  })
})
