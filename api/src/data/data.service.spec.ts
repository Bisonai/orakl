import { Test, TestingModule } from '@nestjs/testing'
import { AggregatorService } from '../aggregator/aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AdapterService } from '../adapter/adapter.service'
import { DataService } from './data.service'
import { PrismaService } from '../prisma.service'

describe('DataService', () => {
  let aggregator: AggregatorService
  let chain: ChainService
  let adapter: AdapterService
  let data: DataService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [DataService, AggregatorService, ChainService, AdapterService, PrismaService]
    }).compile()

    data = module.get<DataService>(DataService)
    aggregator = module.get<AggregatorService>(AggregatorService)
    adapter = module.get<AdapterService>(AdapterService)
    chain = module.get<ChainService>(ChainService)
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
        name: 'Binance-BTC-USD',
        latestRound: -1,
        definition: JSON.stringify({
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
        })
      }
    ]
    const adapterObj = await adapter.create({
      adapterId: 'adapterId-data-test',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    // Aggregator
    const aggregatorData = {
      aggregatorId: 'aggregatorId-data-test',
      active: false,
      name: 'ETH-USD',
      heartbeat: 10_000,
      threshold: 0.04,
      absoluteThreshold: 0.1,
      adapterId: adapterObj.adapterId,
      chainName: chainObj.name
    }
    const aggregatorObj = await aggregator.create(aggregatorData)

    // Data
    const { feeds: feedsObj } = await adapter.findOne({ id: adapterObj.id })
    expect(feeds.length).toBe(1)
    const dataObj = await data.create({
      round: 123,
      timestamp: new Date(Date.now()),
      value: 1,
      aggregator: aggregatorObj.id,
      feed: feedsObj[0].id
    })

    // Cleanup
    await data.remove({ id: dataObj.id })
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
