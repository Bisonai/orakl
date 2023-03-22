import { Test, TestingModule } from '@nestjs/testing'
import { AggregateService } from './aggregate.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('AggregateService', () => {
  let service: AggregateService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [AggregateService, PrismaService]
    }).compile()

    service = module.get<AggregateService>(AggregateService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
