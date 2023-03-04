import { Test, TestingModule } from '@nestjs/testing'
import { ChainService } from './chain.service'
import { PrismaService } from '../prisma.service'

describe('ChainService', () => {
  let service: ChainService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ChainService, PrismaService]
    }).compile()

    service = module.get<ChainService>(ChainService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })

  it('should insert new chain', async () => {
    const name = 'baobab'
    const chain = await service.create({ name })
    expect(chain.name).toBe(name)

    // The same chain cannot be defined twice
    expect(async () => {
      await service.create({ name })
    }).rejects.toThrow()
  })

  it('should update the name of chain', async () => {
    const wrongName = 'cipress'
    const wrongChain = await service.create({ name: wrongName })

    const name = 'cypress'
    const id = wrongChain.id

    const chain = await service.update({
      where: { id },
      chainDto: { name }
    })
    expect(chain.name).toBe(name)
  })

  it('should update the name of chain', async () => {
    const chains = await service.findAll({})
  })
})
