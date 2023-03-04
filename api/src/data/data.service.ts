import { Injectable } from '@nestjs/common'
import { Data, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { CreateDatumDto } from './dto/create-datum.dto'

@Injectable()
export class DataService {
  constructor(private prisma: PrismaService) {}

  create(createDatumDto: CreateDatumDto) {
    return 'This action adds a new datum'
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.DataWhereUniqueInput
    where?: Prisma.DataWhereInput
    orderBy?: Prisma.DataOrderByWithRelationInput
  }): Promise<Data[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.data.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(dataWhereUniqueInput: Prisma.DataWhereUniqueInput): Promise<Data | null> {
    return this.prisma.data.findUnique({
      where: dataWhereUniqueInput
    })
  }
}
