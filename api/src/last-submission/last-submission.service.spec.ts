import { Test, TestingModule } from '@nestjs/testing'
import { LastSubmissionService } from './last-submission.service'

describe('LastSubmissionService', () => {
  let service: LastSubmissionService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [LastSubmissionService]
    }).compile()

    service = module.get<LastSubmissionService>(LastSubmissionService)
  })

  it('should be defined', () => {
    expect(service).toBeDefined()
  })
})
