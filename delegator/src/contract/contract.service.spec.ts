import { Test, TestingModule } from '@nestjs/testing'
import { ContractService } from './contract.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ContractService', () => {
  let contract: ContractService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ContractService, PrismaService]
    }).compile()

    contract = module.get<ContractService>(ContractService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should be defined', () => {
    expect(contract).toBeDefined()
  })

  it('should insert new Contract', async () => {
    const address = '0x0000000000000000000000000000000000000001'
    const contractData = await contract.create({
      address,
      allowAllFunctions: false
    })
    expect(contractData.address).toBe(address)

    // The same Contract cannot be defined twice
    await expect(async () => {
      await contract.create({
        address,
        allowAllFunctions: false
      })
    }).rejects.toThrow()

    // Cleanup
    await contract.remove({ id: contractData.id })
  })
})
