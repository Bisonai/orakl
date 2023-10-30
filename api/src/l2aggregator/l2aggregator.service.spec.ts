import { Test, TestingModule } from '@nestjs/testing'
import { L2aggregatorService } from './l2aggregator.service'
import { PrismaService } from '../prisma.service'

describe('L2aggregatorService', () => {
  let service: L2aggregatorService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [L2aggregatorService, PrismaService]
    }).compile()

    service = module.get<L2aggregatorService>(L2aggregatorService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
