import { Test, TestingModule } from '@nestjs/testing'
import { FeedService } from './feed.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('FeedService', () => {
  let service: FeedService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [FeedService, PrismaService]
    }).compile()

    service = module.get<FeedService>(FeedService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterAll(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
