import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { RedisService } from '../redis.service'
import { AggregateDto } from './dto/aggregate.dto'
import { LatestAggregateDto, LatestAggregateByIdDto } from './dto/latest-aggregate.dto'

@Injectable()
export class AggregateService {
  constructor(
    private prisma: PrismaService,
    private readonly redis: RedisService
  ) {}

  async create(aggregateDto: AggregateDto) {
    const _timestamp = new Date(aggregateDto.timestamp)
    const data: Prisma.AggregateUncheckedCreateInput = {
      timestamp: _timestamp,
      value: aggregateDto.value,
      aggregatorId: BigInt(aggregateDto.aggregatorId)
    }

    await this.redis.set(
      `latestAggregate:${BigInt(aggregateDto.aggregatorId).toString()}`,
      `${_timestamp.toISOString()}|${aggregateDto.value.toString()}`
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

  async findLatestByAggregatorId(latestAggregateByIdDto: LatestAggregateByIdDto) {
    const { aggregatorId } = latestAggregateByIdDto

    const redisKey = `latestAggregate:${aggregatorId.toString()}`
    const result = await this.redis.get(redisKey)

    if (!result) {
      return await this.findLatestByAggregatorIdFromPrisma(aggregatorId)
    }
    const [timestamp, value] = result.split('|')
    return { timestamp, value: BigInt(value) }
  }

  async findLatestByAggregatorIdFromPrisma(aggregatorId) {
    const prismaResult = await this.prisma.aggregate.findFirst({
      where: { aggregatorId },
      orderBy: { timestamp: 'desc' }
    })

    const { timestamp, value } = prismaResult
    return { timestamp, value }
  }
}
