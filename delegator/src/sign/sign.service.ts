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
      process.env.DELEGATOR_PRIVATE_KEY
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
    const signedRawTransaction = await this.updateSignedRawTransaction(transaction.id, signedRawTx)
    return signedRawTransaction
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

  async updateSignedRawTransaction(id: bigint, signedRawTx: string) {
    return this.prisma.transaction.update({
      data: { signedRawTx },
      where: { id }
    })
  }

  async remove(where: Prisma.TransactionWhereUniqueInput) {
    return this.prisma.transaction.delete({
      where
    })
  }

  encryptMethodName(methodName: string) {
    return this.caver.klay.abi.encodeFunctionSignature(methodName)
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

  validateTransaction(tx) {
    // FIXME remove simple whiteListing settings and setup db
    const contractList = {
      '0x93120927379723583c7a0dd2236fcb255e96949f': {
        methods: ['increment()', 'decrement()'],
        reporters: ['0x260836ac4f046b6887bbe16b322e7f1e5f9a0452']
      }
    }

    if (!(tx.to in contractList)) {
      throw new DelegatorError(DelegatorErrorCode.InvalidContract)
    }
    if (!contractList[tx.to].reporters.includes(tx.from)) {
      throw new DelegatorError(DelegatorErrorCode.InvalidReporter)
    }
    const isAllowedMethod = contractList[tx.to].methods.some(
      (method) => this.encryptMethodName(method) == tx.input
    )
    if (!isAllowedMethod) {
      throw new DelegatorError(DelegatorErrorCode.InvalidMethod)
    }
  }
}
