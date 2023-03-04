import { Injectable } from '@nestjs/common'
import { Aggregator, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { CreateAggregatorDto } from './dto/create-aggregator.dto'
import { UpdateAggregatorDto } from './dto/update-aggregator.dto'

@Injectable()
export class AggregatorService {
  constructor(private prisma: PrismaService) {}

  create(createAggregatorDto: CreateAggregatorDto): Promise<Aggregator> {
    const data = {
      aggregatorId: '',
      active: false,
      name: '',
      heartbeat: 10_000,
      threshold: 0.04,
      absoluteThreshold: 0.1,
      adapterId: 1,
      chainId: 1
    }

    return this.prisma.aggregator.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.AggregatorWhereUniqueInput
    where?: Prisma.AggregatorWhereInput
    orderBy?: Prisma.AggregatorOrderByWithRelationInput
  }): Promise<Aggregator[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.aggregator.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(
    aggregatorWhereUniqueInput: Prisma.AggregatorWhereUniqueInput
  ): Promise<Aggregator | null> {
    return this.prisma.aggregator.findUnique({
      where: aggregatorWhereUniqueInput
    })
  }
}
