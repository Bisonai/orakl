import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import type { RedisClientType } from '@redis/client'
import { AdapterService } from '../adapter/adapter.service'
import { AggregatorService } from '../aggregator/aggregator.service'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from '../prisma.service'
import { RedisService } from '../redis.service'
import { AggregateService } from './aggregate.service'

describe('AggregateService', () => {
  let aggregator: AggregatorService
  let chain: ChainService
  let adapter: AdapterService
  let aggregate: AggregateService
  let prisma
  let redis

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        AdapterService,
        AggregatorService,
        ChainService,
        AggregateService,
        PrismaService,
        RedisService
      ]
    }).compile()

    aggregator = module.get<AggregatorService>(AggregatorService)
    adapter = module.get<AdapterService>(AdapterService)
    aggregate = module.get<AggregateService>(AggregateService)
    chain = module.get<ChainService>(ChainService)
    prisma = module.get<PrismaClient>(PrismaService)
    redis = module.get<RedisClientType>(RedisService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(aggregate).toBeDefined()
  })

  it('should insert aggregate', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'aggregator-test-chain' })

    // Adapter
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
    const adapterObj = await adapter.create({
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
    const aggregatorObj = await aggregator.create(aggregatorData)

    // Aggregate
    const aggregateData = {
      aggregatorId: aggregatorObj.id,
      timestamp: new Date().toISOString(),
      value: 10
    }
    const aggregateObj = await aggregate.create(aggregateData)

    // Cleanup
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })
    await aggregate.remove({ id: aggregateObj.id })
  }, 10000)
})
