import { Test, TestingModule } from '@nestjs/testing'
import { AggregatorService } from './aggregator.service'
import { ChainService } from '../chain/chain.service'
import { AdapterService } from '../adapter/adapter.service'
import { PrismaService } from '../prisma.service'

describe('AggregatorService', () => {
  let aggregator: AggregatorService
  let chain: ChainService
  let adapter: AdapterService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AggregatorService, ChainService, AdapterService, PrismaService]
    }).compile()

    aggregator = module.get<AggregatorService>(AggregatorService)
    adapter = module.get<AdapterService>(AdapterService)
    chain = module.get<ChainService>(ChainService)
  })

  it('should be defined', () => {
    expect(aggregator).toBeDefined()
  })

  it('should create aggregator', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'aggregator-test-chain' })

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
      adapterId: '0x0db2dce17745882ea457e651adf1eb0080b5f432d1876df56f1967b5288f338b',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    const data = {
      aggregatorId: '0xd6fbe30bd6249b3093ee065496115e5736bbe760cadfc85598ef27eb4739a849',
      active: false,
      name: 'ETH-USD',
      heartbeat: 10_000,
      threshold: 0.04,
      absoluteThreshold: 0.1,
      adapterId: adapterObj.adapterId,
      chainName: chainObj.name
    }

    const aggregatorObj = await aggregator.create(data)

    // Cleanup
    await aggregator.remove({ id: aggregatorObj.id })
    await adapter.remove({ id: adapterObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
