import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ContractDto } from './dto/contract.dto'
import { ContractConnectDto } from './dto/contract-connect.dto'

@Injectable()
export class ContractService {
  constructor(private prisma: PrismaService) {}

  async create(data: ContractDto) {
    return await this.prisma.contract.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ContractWhereUniqueInput
    where?: Prisma.ContractWhereInput
    orderBy?: Prisma.ContractOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    const contracts = await this.prisma.contract.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy,
      select: {
        id: true,
        address: true,
        allowAllFunctions: true,
        reporter: { select: { address: true } },
        function: { select: { encodedName: true } }
      }
    })
    return contracts
  }

  async findOne(contractWhereUniqueInput: Prisma.ContractWhereUniqueInput) {
    return await this.prisma.contract.findUnique({
      where: contractWhereUniqueInput,
      select: {
        id: true,
        address: true,
        allowAllFunctions: true,
        reporter: { select: { address: true } },
        function: { select: { encodedName: true } }
      }
    })
  }

  async update(params: { where: Prisma.ContractWhereUniqueInput; contractDto: ContractDto }) {
    const { where, contractDto } = params
    return await this.prisma.contract.update({
      data: contractDto,
      where
    })
  }

  async connectReporter(data: ContractConnectDto) {
    await this.prisma.contract.update({
      where: {
        id: data.contractId
      },
      data: {
        reporter: {
          connect: {
            id: data.reporterId
          }
        }
      }
    })
  }

  async disconnectReporter(data: ContractConnectDto) {
    await this.prisma.contract.update({
      where: {
        id: data.contractId
      },
      data: {
        reporter: {
          disconnect: {
            id: data.reporterId
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
