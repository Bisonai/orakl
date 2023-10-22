import { Injectable } from '@nestjs/common'
import { LastSubmissionDto } from './dto/last-submission.dto'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'

@Injectable()
export class LastSubmissionService {
  constructor(private prisma: PrismaService) {}

  async create(lastSubmissionDto: LastSubmissionDto) {
    const data: Prisma.LastSubmissionUncheckedCreateInput = {
      timestamp: new Date(lastSubmissionDto.timestamp),
      value: lastSubmissionDto.value,
      aggregatorId: BigInt(lastSubmissionDto.aggregatorId)
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
    return await this.prisma.lastSubmission.update({
      data: lastSubmissionDto,
      where
    })
  }

  async upsert(data: LastSubmissionDto) {
    const id = 1
    Object.assign(data, { id })
    return await this.prisma.lastSubmission.upsert({
      where: { id },
      create: data,
      update: data
    })
  }

  // /*
  //  * `findLatest` is used by Aggregator heartbeat process that
  //  * periodically requests the latest aggregated data.
  //  */
  // async findLatest(data: LatestSubmittionDto) {
  //   const { aggregatorHash } = data
  //   return await this.prisma.lastSubmission.findFirst({
  //     where: { aggregator: { aggregatorHash } },
  //     orderBy: [
  //       {
  //         timestamp: 'desc'
  //       }
  //     ]
  //   })
  // }

  async remove(where: Prisma.AggregateWhereUniqueInput) {
    return await this.prisma.lastSubmission.delete({
      where
    })
  }
}
