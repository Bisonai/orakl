import { Test, TestingModule } from '@nestjs/testing'
import { ReporterService } from './reporter.service'
import { ChainService } from '../chain/chain.service'
import { ServiceService } from '../service/service.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ReporterService', () => {
  let chain: ChainService
  let service: ServiceService
  let reporter: ReporterService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ReporterService, ServiceService, ChainService, PrismaService]
    }).compile()

    chain = module.get<ChainService>(ChainService)
    service = module.get<ServiceService>(ServiceService)
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

  it('should insert reporter', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'reporter-test-chain' })

    // Service
    const serviceObj = await service.create({ name: 'reporter-test-service' })

    // Reporter
    const reporterObj = await reporter.create({
      address: '0x',
      privateKey: '0x',
      oracleAddress: '0x',
      chain: chainObj.name,
      service: serviceObj.name
    })

    // Cleanup
    await reporter.remove({ id: reporterObj.id })
    await service.remove({ id: serviceObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
