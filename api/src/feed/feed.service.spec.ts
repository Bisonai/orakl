import { Test, TestingModule } from '@nestjs/testing'
import { FeedService } from './feed.service'
import { PrismaService } from '../prisma.service'

describe('FeedService', () => {
  let service: FeedService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [FeedService, PrismaService]
    }).compile()

    service = module.get<FeedService>(FeedService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
