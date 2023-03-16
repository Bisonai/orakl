import { Injectable, HttpException, HttpStatus, Logger } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AdapterDto } from './dto/adapter.dto'
import { PRISMA_ERRORS } from '../errors'
import { ApiAdapterError, ApiAdapterErrorCode } from './adapter-errors'
import { ethers } from 'ethers'
@Injectable()
export class AdapterService {
  private readonly logger = new Logger(AdapterService.name)

  constructor(private prisma: PrismaService) {}

  async create(adapterDto: AdapterDto) {
    const data: Prisma.AdapterCreateInput = {
      adapterHash: adapterDto.adapterHash,
      name: adapterDto.name,
      decimals: adapterDto.decimals,
      feeds: {
        create: adapterDto.feeds
      }
    }

    try {
      await this.computeAdapterHash({ data: adapterDto, verify: true })
    } catch (e) {
      const msg = e.name
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }

    try {
      return await this.prisma.adapter.create({ data })
    } catch (e) {
      const msg = PRISMA_ERRORS[e.code](e.meta)
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.AdapterWhereUniqueInput
    where?: Prisma.AdapterWhereInput
    orderBy?: Prisma.AdapterOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.adapter.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(adapterWhereUniqueInput: Prisma.AdapterWhereUniqueInput) {
    return await this.prisma.adapter.findUnique({
      where: adapterWhereUniqueInput,
      include: {
        feeds: true
      }
    })
  }

  async remove(where: Prisma.AdapterWhereUniqueInput) {
    return await this.prisma.adapter.delete({
      where
    })
  }

  async computeAdapterHash({
    data,
    verify
  }: {
    data: AdapterDto
    verify?: boolean
  }): Promise<AdapterDto> {
    const input = JSON.parse(JSON.stringify(data))

    // Don't use following properties in computation of hash
    delete input.adapterHash

    const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))
    if (verify && data.adapterHash != hash) {
      throw new ApiAdapterError(
        ApiAdapterErrorCode.UnmatchingHash,
        `Hashes do not match!\nExpected ${hash}, received ${data.adapterHash}.`
      )
    } else {
      data.adapterHash = hash
      return data
    }
  }
}
