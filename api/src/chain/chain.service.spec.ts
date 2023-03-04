import { Test, TestingModule } from '@nestjs/testing'
import { ChainService } from './chain.service'
import { PrismaService } from '../prisma.service'

describe('ChainService', () => {
  let chain: ChainService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ChainService, PrismaService]
    }).compile()

    chain = module.get<ChainService>(ChainService)
  })

  it('should be defined', () => {
    expect(chain).toBeDefined()
  })

  it('should insert new chain', async () => {
    const name = 'baobab'
    const ch = await chain.create({ name })
    expect(ch.name).toBe(name)

    // The same chain cannot be defined twice
    expect(async () => {
      await chain.create({ name })
    }).rejects.toThrow()

    // Cleanup
    await chain.remove({ id: ch.id })
  })

  it('should update the name of chain', async () => {
    const wrongName = 'cipress'
    const wrongChain = await chain.create({ name: wrongName })

    const name = 'cypress'
    const id = wrongChain.id

    const ch = await chain.update({
      where: { id },
      chainDto: { name }
    })
    expect(ch.name).toBe(name)

    // Cleanup
    await chain.remove({ id })
  })

  it('should update the name of chain', async () => {
    await chain.findAll({})
  })
})
