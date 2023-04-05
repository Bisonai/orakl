import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { ReporterDto } from './dto/reporter.dto'

@Injectable()
export class ReporterService {
  constructor(private prisma: PrismaService) {}

  async create(reporterDto: ReporterDto) {
    const data: Prisma.ReporterUncheckedCreateInput = {
      address: reporterDto.address,
      organizationId: reporterDto.organizationId
    }
    return await this.prisma.reporter.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ReporterWhereUniqueInput
    where?: Prisma.ReporterWhereInput
    orderBy?: Prisma.ReporterOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.reporter.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy,
      select: {
        id: true,
        address: true,
        contract: { select: { address: true } }
      }
    })
  }

  async findOne(reporterWhereUniqueInput: Prisma.ReporterWhereUniqueInput) {
    return await this.prisma.reporter.findUnique({
      where: reporterWhereUniqueInput,
      select: {
        id: true,
        address: true,
        contract: { select: { address: true } }
      }
    })
  }

  async update(params: { where: Prisma.ReporterWhereUniqueInput; reporterDto: ReporterDto }) {
    const { where, reporterDto } = params
    return await this.prisma.reporter.update({
      data: reporterDto,
      where
    })
  }

  async remove(where: Prisma.ReporterWhereUniqueInput) {
    return await this.prisma.reporter.delete({
      where
    })
  }

  async connectToContract(params: { contractId: bigint; reporterId: bigint }) {
    const { contractId, reporterId } = params
    await this.prisma.reporter.update({
      where: {
        id: reporterId
      },
      data: {
        contract: {
          connect: {
            id: contractId
          }
        }
      }
    })
  }
}
