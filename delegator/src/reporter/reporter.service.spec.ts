import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ReporterService } from './reporter.service'

describe('ReporterService', () => {
  let reporter: ReporterService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ReporterService, PrismaService]
    }).compile()

    reporter = module.get<ReporterService>(ReporterService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(reporter).toBeDefined()
  })
})
