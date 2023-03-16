import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ServiceDto } from './dto/service.dto'

@Injectable()
export class ServiceService {
  constructor(private prisma: PrismaService) {}

  async create(serviceDto: ServiceDto) {
    return await this.prisma.service.create({ data: serviceDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ServiceWhereUniqueInput
    where?: Prisma.ServiceWhereInput
    orderBy?: Prisma.ServiceOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.service.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(serviceWhereUniqueInput: Prisma.ServiceWhereUniqueInput) {
    return await this.prisma.service.findUnique({
      where: serviceWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.ServiceWhereUniqueInput; serviceDto: ServiceDto }) {
    const { where, serviceDto } = params
    return await this.prisma.service.update({
      data: serviceDto,
      where
    })
  }

  async remove(where: Prisma.ServiceWhereUniqueInput) {
    return await this.prisma.service.delete({
      where
    })
  }
}
