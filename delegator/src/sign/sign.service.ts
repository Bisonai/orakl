import { HttpException, HttpStatus, Injectable } from '@nestjs/common'
import { Transaction, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { SignDto } from './dto/sign.dto'
import Caver from 'caver-js'
import { SignatureData } from 'caver-js'
import { DelegatorError, DelegatorErrorCode } from './errors'

@Injectable()
export class SignService {
  caver: any
  feePayerKeyring: any

  constructor(private prisma: PrismaService) {
    this.caver = new Caver(process.env.PROVIDER_URL)
    this.feePayerKeyring = this.caver.wallet.keyring.createFromPrivateKey(
      process.env.DELEGATOR_FEEPAYER_PK
    )
    this.caver.wallet.add(this.feePayerKeyring)
  }

  async create(data: SignDto) {
    const transaction = await this.prisma.transaction.create({ data })
    try {
      this.validateTransaction(transaction)
    } catch (e) {
      const msg = `DelegatorError: ${e.name}`
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }
    const signedRawTx = await this.signTxByFeePayer(transaction)
    const signedTransaction = await this.updateTransaction(transaction, signedRawTx)
    return signedTransaction
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

  async findOne(
    transactionWhereQuniqueInput: Prisma.TransactionWhereUniqueInput
  ): Promise<Transaction | null> {
    return this.prisma.transaction.findUnique({
      where: transactionWhereQuniqueInput
    })
  }

  async update(params: { where: Prisma.TransactionWhereUniqueInput; signDto: SignDto }) {
    const { where, signDto } = params
    return this.prisma.transaction.update({
      data: signDto,
      where
    })
  }

  async updateTransaction(transaction: Transaction, signedRawTx: string) {
    const id = transaction.id
    const succeed = true
    const contract = await this.prisma.contract.findUnique({
      where: {
        address: transaction.to
      }
    })
    const encodedName = transaction.input.substring(0, 10)
    const functions = await this.prisma.function.findUnique({
      where: {
        encodedName
      }
    })
    const reporter = await this.prisma.reporter.findUnique({
      where: {
        address: transaction.from
      }
    })
    return this.prisma.transaction.update({
      data: {
        succeed,
        signedRawTx,
        functionId: functions.id,
        contractId: contract.id
      },
      where: { id }
    })
  }

  async remove(where: Prisma.TransactionWhereUniqueInput) {
    return this.prisma.transaction.delete({
      where
    })
  }

  async signTxByFeePayer(input: Transaction) {
    const signature: SignatureData = new this.caver.wallet.keyring.signatureData([
      input.v,
      input.r,
      input.s
    ])

    const tx = await this.caver.transaction.feeDelegatedSmartContractExecution.create({
      ...input,
      signatures: [signature]
    })
    await this.caver.wallet.signAsFeePayer(this.feePayerKeyring.address, tx)
    return tx.getRawTransaction()
  }

  async validateTransaction(tx) {
    // const result = await this.prisma.contract.findMany({
    //   where: {
    //     address: tx.to,
    //     Reporter: {
    //       some: {
    //         address: tx.from
    //       }
    //     },
    //     Function: {
    //       some: {
    //         encodedName: tx.input.substring(0, 10)
    //       }
    //     }
    //   }
    // })
    // if (result.length == 0) {
    //   throw new DelegatorError(DelegatorErrorCode.NotApprovedTransaction)
    // }
  }
}
