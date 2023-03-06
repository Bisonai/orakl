import { Injectable } from '@nestjs/common'
import { Transaction, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { SignDto } from './dto/sign.dto'

@Injectable()
export class SignService {
  constructor(private prisma: PrismaService) {}

  create(signDto: SignDto): Promise<Transaction> {
    return this.prisma.transaction.create({ data: signDto })
  }

  async findAll(params: {
    skip?: number
    take?: number
    cursor?: Prisma.TransactionWhereUniqueInput
    where?: Prisma.TransactionWhereInput
    orderBy?: Prisma.TransactionOrderByWithRelationInput
  }): Promise<Transaction[]> {
    const { skip, take, cursor, where, orderBy } = params
    return this.prisma.transaction.findMany({
      skip,
      take,
      cursor,
      where,
      orderBy
    })
  }

  findOne(id: number) {
    return `This action returns a #${id} sign`
  }

  // update(id: number, updateSignDto: UpdateSignDto) {
  //   return `This action updates a #${id} sign`
  // }
}
