import { Test, TestingModule } from '@nestjs/testing'
import { ProxyService } from './proxy.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ProxyService', () => {
  let proxy: ProxyService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ProxyService, PrismaService]
    }).compile()

    proxy = module.get<ProxyService>(ProxyService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(proxy).toBeDefined()
  })

  it('should insert new proxy', async () => {
    const proxyData = {
      protocol: 'http',
      host: '127.0.0.1',
      port: 80,
      location: 'kr'
    }
    const proxyObj = await proxy.create(proxyData)
    expect(proxyObj.protocol).toBe(proxyData.protocol)
    expect(proxyObj.host).toBe(proxyData.host)
    expect(proxyObj.port).toBe(proxyData.port)
    expect(proxyObj.location).toBe(proxyData.location)

    // The same proxy cannot be defined twice
    await expect(async () => {
      await proxy.create(proxyData)
    }).rejects.toThrow()

    // Cleanup
    await proxy.remove({ id: proxyObj.id })
  })
})
