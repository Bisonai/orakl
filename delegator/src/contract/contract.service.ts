import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ContractDto } from './dto/contract.dto'

@Injectable()
export class ContractService {
  constructor(private prisma: PrismaService) {}

  async create(contractDto: ContractDto) {
    return await this.prisma.contract.create({
      data: {
        address: contractDto.address
      }
    })
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
      orderBy,
      select: {
        id: true,
        address: true,
        reporter: true
      }
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

  async connectToReporter(params: { contractId: bigint; reporterId: bigint }) {
    const { contractId, reporterId } = params
    await this.prisma.contract.update({
      where: {
        id: contractId
      },
      data: {
        reporter: {
          connect: {
            id: reporterId
          }
        }
      }
    })
  }
  async remove(where: Prisma.ContractWhereUniqueInput) {
    return await this.prisma.contract.delete({
      where
    })
  }
}
