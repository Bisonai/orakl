import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ChainDto } from './dto/chain.dto'

@Injectable()
export class ChainService {
  constructor(private prisma: PrismaService) {}

  async create(chainDto: ChainDto) {
    return await this.prisma.chain.create({ data: chainDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ChainWhereUniqueInput
    where?: Prisma.ChainWhereInput
    orderBy?: Prisma.ChainOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.chain.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(chainWhereUniqueInput: Prisma.ChainWhereUniqueInput) {
    return await this.prisma.chain.findUnique({
      where: chainWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.ChainWhereUniqueInput; chainDto: ChainDto }) {
    const { where, chainDto } = params
    return await this.prisma.chain.update({
      data: chainDto,
      where
    })
  }

  async remove(where: Prisma.ChainWhereUniqueInput) {
    return await this.prisma.chain.delete({
      where
    })
  }
}
