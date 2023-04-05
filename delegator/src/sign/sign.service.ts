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
    const succeed = true
    const contract = await this.prisma.contract.findUnique({
      where: { address: transaction.to }
    })
    const reporter = await this.prisma.reporter.findUnique({
      where: { address: transaction.from }
    })

    const encodedName = transaction.input.substring(0, 10)
    const functions = await this.prisma.function.findUnique({
      where: { encodedName }
    })

    const data: SignDto = { ...transaction }
    data.succeed = succeed
    data.signedRawTx = signedRawTx
    data.reporterId = reporter.id
    data.contractId = contract.id
    data.functionId = functions ? functions.id : null

    return this.prisma.transaction.update({
      data,
      where: { id: transaction.id }
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
    const contract = await this.prisma.contract.findUnique({
      where: { address: tx.to },
      include: {
        reporter: { select: { address: true } },
        function: { select: { encodedName: true } }
      }
    })

    // extract reporters address
    const reporters = contract.reporter.map((i) => {
      return i.address
    })
    // extract function encodedName address
    const functions = contract.function.map((i) => {
      return i.encodedName
    })

    if (
      contract &&
      reporters.includes(tx.from) &&
      (contract.allowAllFunctions || functions.includes(tx.input.substring(0, 10)))
    ) {
      return
    } else {
      throw new DelegatorError(DelegatorErrorCode.NotApprovedTransaction)
    }
  }
}
