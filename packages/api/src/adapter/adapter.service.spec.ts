import { Test, TestingModule } from '@nestjs/testing'
import { AdapterService } from './adapter.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('AdapterService', () => {
  let adapter: AdapterService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AdapterService, PrismaService]
    }).compile()

    adapter = module.get<AdapterService>(AdapterService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(adapter).toBeDefined()
  })

  it('should insert adapter and find it', async () => {
    const feeds = [
      {
        name: 'Binance-BTC-USD-adapter',
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

    const { id } = await adapter.create({
      adapterHash: '0xbb555a249d01133784fa04c608ce03c129f73f2a1ef7473d0cfffdc4bcba794e',
      name: 'BTC-USD',
      decimals: 8,
      feeds
    })

    const adapterObj = await adapter.findOne({ id })
    expect(adapterObj.feeds.length).toBe(1)

    // Cleanup
    await adapter.remove({ id })
  })
})
