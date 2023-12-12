import { Inject, Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import type { RedisClientType } from 'redis'
import { PrismaService } from '../prisma.service'
import { AggregateDto } from './dto/aggregate.dto'
import { LatestAggregateDto } from './dto/latest-aggregate.dto'

@Injectable()
export class AggregateService {
  constructor(
    @Inject('REDIS_CLIENT')
    private prisma: PrismaService,
    private readonly redis: RedisClientType
  ) {}

  async create(aggregateDto: AggregateDto) {
    const data: Prisma.AggregateUncheckedCreateInput = {
      timestamp: new Date(aggregateDto.timestamp),
      value: aggregateDto.value,
      aggregatorId: BigInt(aggregateDto.aggregatorId)
    }

    this.redis.set(
      `latestAggregate:${BigInt(aggregateDto.aggregatorId).toString()}`,
      aggregateDto.value.toString()
    )

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

  /*
   * `findLatest` is used by Aggregator heartbeat process that
   * periodically requests the latest aggregated data.
   */
  async findLatest(latestAggregateDto: LatestAggregateDto) {
    const { aggregatorHash } = latestAggregateDto

    const query = Prisma.sql`SELECT aggregate_id as id, timestamp, value, aggregator_id as "aggregatorId"
      FROM aggregates
      WHERE aggregator_id = (SELECT aggregator_id FROM aggregators WHERE aggregator_hash = ${aggregatorHash})
      ORDER BY timestamp DESC
      LIMIT 1;`
    const result: Prisma.AggregateScalarFieldEnum[] = await this.prisma.$queryRaw(query)
    if (result && result.length == 1) {
      return result[0]
    } else {
      return null
    }
  }
}
