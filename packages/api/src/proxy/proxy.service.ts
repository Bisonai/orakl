import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ProxyDto } from './dto/proxy'

@Injectable()
export class ProxyService {
  constructor(private prisma: PrismaService) {}

  async create(proxyDto: ProxyDto) {
    return await this.prisma.proxy.create({ data: proxyDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ProxyWhereUniqueInput
    where?: Prisma.ProxyWhereInput
    orderBy?: Prisma.ProxyOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.proxy.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(proxyWhereUniqueInput: Prisma.ProxyWhereUniqueInput) {
    return await this.prisma.proxy.findUnique({
      where: proxyWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.ProxyWhereUniqueInput; proxyDto: ProxyDto }) {
    const { where, proxyDto } = params
    return await this.prisma.proxy.update({
      data: proxyDto,
      where
    })
  }

  async remove(where: Prisma.ProxyWhereUniqueInput) {
    return await this.prisma.proxy.delete({
      where
    })
  }
}
