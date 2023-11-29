import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import { ChainService } from '../chain/chain.service'
import { PrismaService } from '../prisma.service'
import { ServiceService } from '../service/service.service'
import { ReporterService } from './reporter.service'

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
    const prk = '0x137388d6b6896346b330b7becd87e0de69bd320ae052c54445e5acfd18e4ff0e'
    const reporterObj = await reporter.create({
      address: '0x',
      privateKey: prk,
      oracleAddress: '0x',
      chain: chainObj.name,
      service: serviceObj.name
    })

    const readReporter = await reporter.findOne({ id: reporterObj.id })
    expect(readReporter.privateKey).toEqual(prk)

    // Cleanup
    await reporter.remove({ id: reporterObj.id })
    await service.remove({ id: serviceObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
