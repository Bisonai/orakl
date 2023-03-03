import { Injectable } from '@nestjs/common'
import { Adapter, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { FeedDto } from '../feed/dto/feed.dto'
import { CreateAdapterDto } from './dto/create-adapter.dto'
import { UpdateAdapterDto } from './dto/update-adapter.dto'

@Injectable()
export class AdapterService {
  constructor(private prisma: PrismaService) {}

  create(adapterDto: CreateAdapterDto): Promise<Adapter> {
    const data: Prisma.AdapterCreateInput = {
      adapterId: adapterDto.adapterId,
      name: adapterDto.name,
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

  async findOne(chainWhereUniqueInput: Prisma.AdapterWhereUniqueInput): Promise<Adapter | null> {
    return this.prisma.adapter.findUnique({
      where: chainWhereUniqueInput
    })
  }

  update(id: number, updateAdapterDto: UpdateAdapterDto) {
    return `This action updates a #${id} adapter`
  }

  remove(id: number) {
    return `This action removes a #${id} adapter`
  }
}
