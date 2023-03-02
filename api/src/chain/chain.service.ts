import { Injectable } from '@nestjs/common'
import { Chain, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'

@Injectable()
export class ChainService {
  constructor(private prisma: PrismaService) {}

  async create(data: Prisma.ChainCreateInput): Promise<Chain> {
    const { name } = data
    return this.prisma.chain.create({
      data
    })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ChainWhereUniqueInput
    where?: Prisma.ChainWhereInput
    orderBy?: Prisma.ChainOrderByWithRelationInput
  }): Promise<Chain[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.chain.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(chainWhereUniqueInput: Prisma.ChainWhereUniqueInput): Promise<Chain | null> {
    return this.prisma.chain.findUnique({
      where: chainWhereUniqueInput
    })
  }

  async update(params: {
    where: Prisma.ChainWhereUniqueInput
    data: Prisma.ChainUpdateInput
  }): Promise<Chain> {
    const { where, data } = params
    return this.prisma.chain.update({
      data,
      where
    })
  }

  async remove(where: Prisma.ChainWhereUniqueInput): Promise<Chain> {
    return this.prisma.chain.delete({
      where
    })
  }
}
