import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregateDto } from './dto/aggregate.dto'

@Injectable()
export class AggregateService {
  constructor(private prisma: PrismaService) {}

  async create(aggregateDto: AggregateDto) {
    const data: Prisma.AggregateUncheckedCreateInput = {
      timestamp: new Date(aggregateDto.timestamp),
      value: aggregateDto.value,
      aggregatorId: aggregateDto.aggregatorId
    }

    return await this.prisma.aggregate.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.AggregateWhereUniqueInput
    where?: Prisma.AggregateWhereInput
    orderBy?: Prisma.AggregateOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.aggregate.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(aggregateWhereUniqueInput: Prisma.AggregateWhereUniqueInput) {
    return await this.prisma.aggregate.findUnique({
      where: aggregateWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.AggregateWhereUniqueInput; aggregateDto: AggregateDto }) {
    const { where, aggregateDto } = params
    return await this.prisma.aggregate.update({
      data: aggregateDto,
      where
    })
  }

  async remove(where: Prisma.AggregateWhereUniqueInput) {
    return await this.prisma.aggregate.delete({
      where
    })
  }
}
