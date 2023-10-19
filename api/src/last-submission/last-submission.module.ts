import { Module } from '@nestjs/common'
import { LastSubmissionService } from './last-submission.service'
import { LastSubmissionController } from './last-submission.controller'

@Module({
  controllers: [LastSubmissionController],
  providers: [LastSubmissionService]
})
export class LastSubmissionModule {}
