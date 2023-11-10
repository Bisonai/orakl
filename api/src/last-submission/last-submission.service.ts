import { Injectable } from '@nestjs/common'
import { LastSubmissionDto } from './dto/last-submission.dto'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregatorDto } from './dto/aggregator.dto'

@Injectable()
export class LastSubmissionService {
  constructor(private prisma: PrismaService) {}

  async upsert(lastSubmissionDto: LastSubmissionDto) {
    const submissionData: Prisma.LastSubmissionUncheckedCreateInput = {
      timestamp: new Date(),
      value: lastSubmissionDto.value,
      aggregatorId: lastSubmissionDto.aggregatorId
    }

    const data: Prisma.LastSubmissionUpsertArgs = {
      where: { aggregatorId: BigInt(submissionData.aggregatorId) },
      create: submissionData,
      update: submissionData
    }

    return await this.prisma.lastSubmission.upsert(data)
  }

  async findByhash(aggregator: AggregatorDto) {
    const { aggregatorHash } = aggregator
    return await this.prisma.lastSubmission.findFirst({
      where: { aggregator: { aggregatorHash } },
      orderBy: [{ timestamp: 'desc' }]
    })
  }

  async remove(where: Prisma.AggregateWhereUniqueInput) {
    return await this.prisma.lastSubmission.delete({
      where
    })
  }
}
