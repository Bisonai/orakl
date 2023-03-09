import { Injectable, HttpStatus, HttpException, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AggregatorDto } from './dto/aggregator.dto'

@Injectable()
export class AggregatorService {
  private readonly logger = new Logger(AggregatorService.name)

  constructor(private prisma: PrismaService) {}

  async create(aggregatorDto: AggregatorDto) {
    // chain
    const chainName = aggregatorDto.chain
    const chain = await this.prisma.chain.findUnique({
      where: { name: chainName }
    })

    if (chain == null) {
      const msg = `chain.name [${chainName}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    // adapter
    const adapterHash = aggregatorDto.adapterHash
    const adapter = await this.prisma.adapter.findUnique({
      where: { adapterHash }
    })

    if (adapter == null) {
      const msg = `adapter.adapterHash [${adapterHash}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    const data: Prisma.AggregatorUncheckedCreateInput = {
      aggregatorHash: aggregatorDto.aggregatorHash,
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
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.aggregator.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findUnique(aggregatorWhereUniqueInput: Prisma.AggregatorWhereUniqueInput) {
    return await this.prisma.aggregator.findUnique({
      where: aggregatorWhereUniqueInput,
      include: {
        adapter: {
          include: {
            feeds: true
          }
        }
      }
    })
  }

  async remove(where: Prisma.AggregatorWhereUniqueInput) {
    return await this.prisma.aggregator.delete({
      where
    })
  }

  async update(params: { where: Prisma.AggregatorWhereUniqueInput; active: boolean }) {
    const { where, active } = params
    return await this.prisma.aggregator.update({
      data: { active },
      where
    })
  }
}
