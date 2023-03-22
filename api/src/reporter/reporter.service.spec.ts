import { Test, TestingModule } from '@nestjs/testing'
import { ReporterService } from './reporter.service'
import { PrismaClient } from '@prisma/client'

describe('ReporterService', () => {
  let reporter: ReporterService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ReporterService]
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
