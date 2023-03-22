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
        name: 'Binance-BTC-USD',
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
      adapterHash: '0x0378fa3bc8d033fe1207d50b4c53e9c2c25b908478160d3dd7869259242e589c',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    // Aggregator
    const aggregatorData = {
      aggregatorHash: '0x6d614e228f67c418878d7b26300ef03d1e79693be1c8c00143b3adcb245b73cc',
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
