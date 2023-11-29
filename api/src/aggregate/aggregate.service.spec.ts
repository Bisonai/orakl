import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregateService } from './aggregate.service'

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
