import { Test, TestingModule } from '@nestjs/testing'
import { AggregatorService } from '../aggregator/aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AdapterService } from '../adapter/adapter.service'
import { DataService } from './data.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('DataService', () => {
  let aggregator: AggregatorService
  let chain: ChainService
  let adapter: AdapterService
  let data: DataService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [DataService, AggregatorService, ChainService, AdapterService, PrismaService]
    }).compile()

    data = module.get<DataService>(DataService)
    aggregator = module.get<AggregatorService>(AggregatorService)
    adapter = module.get<AdapterService>(AdapterService)
    chain = module.get<ChainService>(ChainService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterAll(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(data).toBeDefined()
  })

  it('should create aggregator', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'data-test-chain' })

    // Adapter
    const feeds = [
      {
        name: 'Binance-BTC-USD-data',
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
      adapterHash: '0x8a259c93fe936ccaa9e7488663209cd8bc0a8467064e4e73d4089394a178279f',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    // Aggregator
    const aggregatorData = {
      aggregatorHash: '0xaf64805e5506307d544b40d244344fd9429b07e4f92a548118081eaaa6a5e0e6',
      active: false,
      name: 'BTC-USD',
      address: '0x222',
      heartbeat: 10_000,
      threshold: 0.04,
      absoluteThreshold: 0.1,
      adapterHash: adapterObj.adapterHash,
      chain: chainObj.name
    }
    const aggregatorObj = await aggregator.create(aggregatorData)

    // Data
    const { feeds: feedsObj } = await adapter.findOne({ id: adapterObj.id })
    expect(feeds.length).toBe(1)
    const dataObj = await data.create({
      timestamp: new Date(Date.now()),
      value: BigInt(2241772466578),
      aggregatorId: aggregatorObj.id,
      feedId: feedsObj[0].id
    })

    // Cleanup
    await data.remove({ id: dataObj.id })
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })
  }, 10000)
})
