import { Injectable } from '@nestjs/common'
import { LastSubmissionDto } from './dto/last-submission.dto'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'

@Injectable()
export class LastSubmissionService {
  constructor(private prisma: PrismaService) {}

  async create(lastSubmissionDto: LastSubmissionDto) {
    const data: Prisma.LastSubmissionUncheckedCreateInput = {
      timestamp: new Date(),
      value: lastSubmissionDto.value,
      aggregatorId: lastSubmissionDto.aggregatorId
    }

    return await this.prisma.lastSubmission.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.LastSubmissionWhereUniqueInput
    where?: Prisma.LastSubmissionWhereInput
    orderBy?: Prisma.LastSubmissionOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.lastSubmission.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(lastSubmissionWhereUniqueInput: Prisma.LastSubmissionWhereUniqueInput) {
    return await this.prisma.lastSubmission.findUnique({
      where: lastSubmissionWhereUniqueInput
    })
  }

  async update(params: {
    where: Prisma.LastSubmissionWhereUniqueInput
    lastSubmissionDto: LastSubmissionDto
  }) {
    const { where, lastSubmissionDto } = params
    const data = {
      timestamp: new Date(),
      value: lastSubmissionDto.value,
      aggregatorId: lastSubmissionDto.aggregatorId
    }

    return await this.prisma.lastSubmission.update({
      data,
      where
    })
  }

  async upsert(lastSubmissionDto: LastSubmissionDto) {
    const submissionData: Prisma.LastSubmissionUncheckedCreateInput = {
      timestamp: new Date(),
      value: lastSubmissionDto.value,
      aggregatorId: lastSubmissionDto.aggregatorId
    }

    const data: Prisma.LastSubmissionUpsertArgs = {
      where: {
        aggregatorId: BigInt(submissionData.aggregatorId)
      },
      create: submissionData,
      update: submissionData
    }

    return await this.prisma.lastSubmission.upsert(data)
  }

  async remove(where: Prisma.AggregateWhereUniqueInput) {
    return await this.prisma.lastSubmission.delete({
      where
    })
  }
}
