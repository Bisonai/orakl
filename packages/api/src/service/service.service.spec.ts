import { Test, TestingModule } from '@nestjs/testing'
import { ServiceService } from './service.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ServiceService', () => {
  let service: ServiceService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ServiceService, PrismaService]
    }).compile()
    service = module.get<ServiceService>(ServiceService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })

  it('should insert new service', async () => {
    const name = 'NewService'
    const s = await service.create({ name })
    expect(s.name).toBe(name)

    // The same service cannot be defined twice
    await expect(async () => {
      await service.create({ name })
    }).rejects.toThrow()

    // Cleanup
    await service.remove({ id: s.id })
  })

  it('should update the name of service', async () => {
    const oldName = 'OldService'
    const oldService = await service.create({ name: oldName })

    const name = 'NewService'
    const id = oldService.id

    const s = await service.update({
      where: { id },
      serviceDto: { name }
    })
    expect(s.name).toBe(name)

    // Cleanup
    await service.remove({ id })
  })
})
