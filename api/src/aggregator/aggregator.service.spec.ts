import { Test, TestingModule } from '@nestjs/testing'
import { AggregatorService } from './aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AdapterService } from '../adapter/adapter.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('AggregatorService', () => {
  let aggregator: AggregatorService
  let chain: ChainService
  let adapter: AdapterService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AggregatorService, ChainService, AdapterService, PrismaService]
    }).compile()

    aggregator = module.get<AggregatorService>(AggregatorService)
    adapter = module.get<AdapterService>(AdapterService)
    chain = module.get<ChainService>(ChainService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(aggregator).toBeDefined()
  })

  it('should insert aggregator', async () => {
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

    // Cleanup
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })
  }, 10000)
})
