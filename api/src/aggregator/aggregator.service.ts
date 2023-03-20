import { Injectable, HttpStatus, HttpException, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { ethers } from 'ethers'
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
      name: aggregatorDto.name,
      address: aggregatorDto.address,
      heartbeat: aggregatorDto.heartbeat,
      threshold: aggregatorDto.threshold,
      absoluteThreshold: aggregatorDto.absoluteThreshold,
      adapterId: adapter.id,
      chainId: chain.id
    }

    try {
      await this.computeAggregatorHash({ data: aggregatorDto, verify: true })
    } catch (e) {
      console.log(e)
      this.logger.error(e)
      throw new HttpException(e, HttpStatus.BAD_REQUEST)
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

  // TODO: FIX Updata function
  // async update(params: { where: Prisma.AggregatorWhereUniqueInput; aggregatorDto: AggregatorDto }) {
  //   const { where, aggregatorDto } = params
  //   return await this.prisma.aggregator.update({
  //     data: aggregatorDto,
  //     where
  //   })
  // }

  async computeAggregatorHash({
    data,
    verify
  }: {
    data: AggregatorDto
    verify?: boolean
  }): Promise<AggregatorDto> {
    const input = JSON.parse(JSON.stringify(data))

    // Don't use following properties in computation of hash
    delete input.aggregatorHash
    delete input.address

    const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

    if (verify && data.aggregatorHash != hash) {
      throw `Hashes do not match!\nExpected ${hash}, received ${data.aggregatorHash}.`
    } else {
      data.aggregatorHash = hash
      return data
    }
  }
}
