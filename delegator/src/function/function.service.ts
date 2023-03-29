import { Injectable } from '@nestjs/common'
import { Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { FunctionDto } from './dto/function.dto'
import Caver from 'caver-js'

@Injectable()
export class FunctionService {
  caver: any

  constructor(private prisma: PrismaService) {
    this.caver = new Caver(process.env.PROVIDER_URL)
  }

  encryptFunctionName(functionName: string) {
    return this.caver.klay.abi.encodeFunctionSignature(functionName)
  }

  async create(functionDto: FunctionDto) {
    const data: Prisma.FunctionUncheckedCreateInput = {
      name: functionDto.name,
      encodedName: this.encryptFunctionName(functionDto.name),
      contractId: BigInt(functionDto.contractId)
    }
    return await this.prisma.function.create({ data })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.FunctionWhereUniqueInput
    where?: Prisma.FunctionWhereInput
    orderBy?: Prisma.FunctionOrderByWithRelationInput
  }) {
    const { skip, take, cursor, where, orderBy } = params
    const functions = await this.prisma.function.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy,
      select: {
        id: true,
        name: true,
        encodedName: true,
        contract: { select: { address: true } }
      }
    })
    return functions.map((M) => {
      return {
        id: M.id,
        name: M.name,
        encodedName: M.encodedName,
        address: M.contract.address
      }
    })
  }

  async findOne(functionWhereUniqueInput: Prisma.FunctionWhereUniqueInput) {
    return await this.prisma.function.findUnique({
      where: functionWhereUniqueInput
    })
  }

  async update(params: { where: Prisma.FunctionWhereUniqueInput; functionDto: FunctionDto }) {
    const { where, functionDto } = params
    return await this.prisma.function.update({
      data: functionDto,
      where
    })
  }

  async remove(where: Prisma.FunctionWhereUniqueInput) {
    return await this.prisma.function.delete({
      where
    })
  }
}
