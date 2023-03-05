import { Injectable } from '@nestjs/common'
import { Aggregator, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregatorDto } from './dto/aggregator.dto'

@Injectable()
export class AggregatorService {
  constructor(private prisma: PrismaService) {}

  async create(aggregatorDto: AggregatorDto): Promise<Aggregator> {
    const chain = await this.prisma.chain.findUnique({
      where: { name: aggregatorDto.chain }
    })

    const adapter = await this.prisma.adapter.findUnique({
      where: { adapterId: aggregatorDto.adapterId }
    })

    const data: Prisma.AggregatorUncheckedCreateInput = {
      aggregatorId: aggregatorDto.id,
      active: aggregatorDto.active,
      name: aggregatorDto.name,
      address: aggregatorDto.address,
      heartbeat: aggregatorDto.heartbeat,
      threshold: aggregatorDto.threshold,
      absoluteThreshold: aggregatorDto.absoluteThreshold,
      adapterId: adapter.id,
      chainId: chain.id
    }

    return await this.prisma.aggregator.create({ data })
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

  async remove(where: Prisma.AggregatorWhereUniqueInput): Promise<Aggregator> {
    return this.prisma.aggregator.delete({
      where
    })
  }
}
