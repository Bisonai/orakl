import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ContractDto } from './dto/contract.dto'

@Injectable()
export class ContractService {
  constructor(private prisma: PrismaService) {}

  async create(contractDto: ContractDto) {
    return await this.prisma.contract.create({ data: contractDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ContractWhereUniqueInput
    where?: Prisma.ContractWhereInput
    orderBy?: Prisma.ContractOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.contract.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(contractWhereUniqueInput: Prisma.ContractWhereUniqueInput) {
    return await this.prisma.contract.findUnique({
      where: contractWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.ContractWhereUniqueInput; contractDto: ContractDto }) {
    const { where, contractDto } = params
    return await this.prisma.contract.update({
      data: contractDto,
      where
    })
  }

  async remove(where: Prisma.ContractWhereUniqueInput) {
    return await this.prisma.contract.delete({
      where
    })
  }
}
