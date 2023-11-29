import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { FunctionService } from './function.service'

describe('FunctionService', () => {
  let functionService: FunctionService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [FunctionService, PrismaService]
    }).compile()

    functionService = module.get<FunctionService>(FunctionService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(functionService).toBeDefined()
  })
})
