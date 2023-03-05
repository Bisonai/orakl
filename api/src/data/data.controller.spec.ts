import { Test, TestingModule } from '@nestjs/testing'
import { DataController } from './data.controller'
import { DataService } from './data.service'
import { PrismaService } from '../prisma.service'

describe('DataController', () => {
  let controller: DataController

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [DataController],
      providers: [DataService, PrismaService]
    }).compile()

    controller = module.get<DataController>(DataController)
  })

  it('should be defined', () => {
    expect(controller).toBeDefined()
  })
})
