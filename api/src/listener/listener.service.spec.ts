import { Test, TestingModule } from '@nestjs/testing'
import { ListenerService } from './listener.service'
import { ChainService } from '../chain/chain.service'
import { ServiceService } from '../service/service.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ListenerService', () => {
  let chain: ChainService
  let service: ServiceService
  let listener: ListenerService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ListenerService, ServiceService, ChainService, PrismaService]
    }).compile()

    chain = module.get<ChainService>(ChainService)
    service = module.get<ServiceService>(ServiceService)
    listener = module.get<ListenerService>(ListenerService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(listener).toBeDefined()
  })

  it('should insert listener', async () => {
    // Chain
    const chainObj = await chain.create({ name: 'listener-test-chain' })

    // Service
    const serviceObj = await service.create({ name: 'listener-test-service' })

    const listenerObj = await listener.create({
      address: '0x',
      eventName: 'TestEventName',
      chain: chainObj.name,
      service: serviceObj.name
    })

    // Cleanup
    await listener.remove({ id: listenerObj.id })
    await service.remove({ id: serviceObj.id })
    await chain.remove({ id: chainObj.id })
  })
})
