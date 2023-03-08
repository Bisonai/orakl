import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregateDto } from './dto/aggregate.dto'

@Injectable()
export class AggregateService {
  constructor(private prisma: PrismaService) {}

  create(aggregateDto: AggregateDto) {
    return 'This action adds a new aggregate'
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.AggregateWhereUniqueInput
    where?: Prisma.AggregateWhereInput
    orderBy?: Prisma.AggregateOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.aggregate.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  findOne(aggregateWhereUniqueInput: Prisma.AggregateWhereUniqueInput) {
    return this.prisma.aggregate.findUnique({
      where: aggregateWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.AggregateWhereUniqueInput; aggregateDto: AggregateDto }) {
    const { where, aggregateDto } = params
    return this.prisma.aggregate.update({
      data: aggregateDto,
      where
    })
  }

  remove(where: Prisma.AggregateWhereUniqueInput) {
    return this.prisma.aggregate.delete({
      where
    })
  }
}
