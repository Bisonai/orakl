import { Injectable } from '@nestjs/common'
import { ErrorDto } from './dto/error.dto'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'

@Injectable()
export class ErrorService {
  constructor(private prisma: PrismaService) {}

  async create(errorDto: ErrorDto) {
    // Error data
    const data: Prisma.ErrorUncheckedCreateInput = {
      requestId: errorDto.requestId,
      timestamp: errorDto.timestamp,
      errorMsg: errorDto.errorMsg
    }
    return await this.prisma.error.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.ErrorWhereUniqueInput
    where?: Prisma.ErrorWhereInput
    orderBy?: Prisma.ErrorOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    return await this.prisma.error.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  async findOne(errorWhereUniqueInput: Prisma.ErrorWhereUniqueInput) {
    return await this.prisma.error.findUnique({
      where: errorWhereUniqueInput
    })
  }
}
