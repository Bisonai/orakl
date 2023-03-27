import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { MethodDto } from './dto/method.dto'

@Injectable()
export class MethodService {
  constructor(private prisma: PrismaService) {}

  async create(methodDto: MethodDto) {
    const data: Prisma.MethodUncheckedCreateInput = {
      name: methodDto.name,
      contractId: methodDto.contractId
    }
    return await this.prisma.method.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.MethodWhereUniqueInput
    where?: Prisma.MethodWhereInput
    orderBy?: Prisma.MethodOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.method.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(methodWhereUniqueInput: Prisma.MethodWhereUniqueInput) {
    return await this.prisma.organization.findUnique({
      where: methodWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.MethodWhereUniqueInput; methodDto: MethodDto }) {
    const { where, methodDto } = params
    return await this.prisma.organization.update({
      data: methodDto,
      where
    })
  }

  async remove(where: Prisma.MethodWhereUniqueInput) {
    return await this.prisma.organization.delete({
      where
    })
  }
}
