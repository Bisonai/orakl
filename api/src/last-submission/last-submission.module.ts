import { Module } from '@nestjs/common'
import { LastSubmissionService } from './last-submission.service'
import { LastSubmissionController } from './last-submission.controller'
import { PrismaService } from '../prisma.service'

@Module({
  controllers: [LastSubmissionController],
  providers: [LastSubmissionService, PrismaService]
})
export class LastSubmissionModule {}
