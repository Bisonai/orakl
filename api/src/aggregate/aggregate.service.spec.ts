import { Test, TestingModule } from '@nestjs/testing'
import { AggregateService } from './aggregate.service'
import { PrismaService } from '../prisma.service'

describe('AggregateService', () => {
  let service: AggregateService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AggregateService, PrismaService]
    }).compile()

    service = module.get<AggregateService>(AggregateService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
