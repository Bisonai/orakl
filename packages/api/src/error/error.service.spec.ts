import { Test, TestingModule } from '@nestjs/testing'
import { ErrorService } from './error.service'
import { PrismaService } from '../prisma.service'
import { PrismaClient } from '@prisma/client'

describe('ErrorService', () => {
  let service: ErrorService
  let prisma

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [ErrorService, PrismaService]
    }).compile()

    service = module.get<ErrorService>(ErrorService)
    prisma = module.get<PrismaClient>(PrismaService)
  })

  afterAll(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('should create new Error', async () => {
    const code = '10020'
    const name = 'MissingKeyInJson'
    const requestId =
      '66649924661314489704239946349158829048302840686075232939396730072454733114998'
    const stack = `
        MissingKeyInJson
            at wrapper (file:///app/dist/worker/reducer.js:19:23)
            at file:///app/dist/utils.js:11:61
            at Array.reduce (<anonymous>)
            at file:///app/dist/utils.js:11:44
            at processRequest (file:///app/dist/worker/request-response.js:58:34)
            at process.processTicksAndRejections (node:internal/process/task_queues:95:5)
            at async Worker.wrapper [as processFn] (file:///app/dist/worker/request-response.js:27:25)
            at async Worker.processJob (/app/node_modules/bullmq/dist/cjs/classes/worker.js:339:28)
            at async Worker.retryIfFailed (/app/node_modules/bullmq/dist/cjs/classes/worker.js:513:24)`

    const errorData = {
      requestId,
      timestamp: new Date(Date.now()),
      code,
      name,
      stack
    }
    const errorObj = await service.create(errorData)
    expect(errorObj).toBeDefined()
  })
})
