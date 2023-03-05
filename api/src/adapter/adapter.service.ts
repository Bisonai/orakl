import { Injectable } from '@nestjs/common'
import { Adapter, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { AdapterDto } from './dto/adapter.dto'

@Injectable()
export class AdapterService {
  constructor(private prisma: PrismaService) {}

  create(adapterDto: AdapterDto): Promise<Adapter> {
    // TODO validate

    const data: Prisma.AdapterCreateInput = {
      adapterId: adapterDto.id,
      name: adapterDto.name,
      decimals: adapterDto.decimals,
      feeds: {
        create: adapterDto.feeds
      }
    }

    return this.prisma.adapter.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.AdapterWhereUniqueInput
    where?: Prisma.AdapterWhereInput
    orderBy?: Prisma.AdapterOrderByWithRelationInput
  }): Promise<Adapter[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.adapter.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(adapterWhereUniqueInput: Prisma.AdapterWhereUniqueInput) {
    return this.prisma.adapter.findUnique({
      where: adapterWhereUniqueInput,
      include: {
        feeds: true
      }
    })
  }

  async remove(where: Prisma.AdapterWhereUniqueInput): Promise<Adapter> {
    return this.prisma.adapter.delete({
      where
    })
  }
}
