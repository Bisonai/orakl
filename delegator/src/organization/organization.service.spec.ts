import { Test, TestingModule } from '@nestjs/testing'
import { PrismaClient } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { OrganizationService } from './organization.service'

describe('OrganizationService', () => {
  let organization: OrganizationService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [OrganizationService, PrismaService]
    }).compile()

    organization = module.get<OrganizationService>(OrganizationService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(organization).toBeDefined()
  })

  it('should insert new Organization', async () => {
    const name = 'Bisonai'
    const or = await organization.create({ name })
    expect(or.name).toBe(name)

    // The same Organization cannot be defined twice
    await expect(async () => {
      await organization.create({ name })
    }).rejects.toThrow()

    // Cleanup
    await organization.remove({ id: or.id })
  })
})
