import { Test, TestingModule } from '@nestjs/testing'
import { AggregatorController } from './aggregator.controller'
import { AggregatorService } from './aggregator.service'
import { PrismaService } from '../prisma.service'

describe('AggregatorController', () => {
  let controller: AggregatorController

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [AggregatorController],
      providers: [AggregatorService, PrismaService]
    }).compile()

    controller = module.get<AggregatorController>(AggregatorController)
  })

  it('should be defined', () => {
    expect(controller).toBeDefined()
  })
})
