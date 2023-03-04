import { Test, TestingModule } from '@nestjs/testing'
import { ChainService } from './chain.service'
import { PrismaService } from '../prisma.service'

describe('ChainService', () => {
  let service: ChainService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ChainService, PrismaService]
    }).compile()

    service = module.get<ChainService>(ChainService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
