import { Test, TestingModule } from '@nestjs/testing'
import { AdapterService } from './adapter.service'
import { PrismaService } from '../prisma.service'

describe('AdapterService', () => {
  let service: AdapterService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AdapterService, PrismaService]
    }).compile()

    service = module.get<AdapterService>(AdapterService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
