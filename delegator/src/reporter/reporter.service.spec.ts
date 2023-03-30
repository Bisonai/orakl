import { Test, TestingModule } from '@nestjs/testing'
import { ReporterService } from './reporter.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

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
