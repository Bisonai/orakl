import { HttpException, HttpStatus, Injectable, Logger } from '@nestjs/common'
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
  private readonly logger = new Logger(SignService.name)

  constructor(private prisma: PrismaService) {
    this.caver = new Caver(process.env.PROVIDER_URL)
  }

  async initialize({ feePayerPrivateKey }: { feePayerPrivateKey?: string }) {
    if (!feePayerPrivateKey) {
      const feePayers: any[] = await this.prisma.$queryRaw`SELECT * FROM fee_payers`
      if (feePayers.length == 0) {
        throw new DelegatorError(DelegatorErrorCode.NoFeePayer)
      } else if (feePayers.length == 1) {
        feePayerPrivateKey = feePayers[0]?.privateKey
      } else {
        throw new DelegatorError(DelegatorErrorCode.TooManyFeePayers)
      }
    }

    this.feePayerKeyring = this.caver.wallet.keyring.createFromPrivateKey(feePayerPrivateKey)
    this.caver.wallet.add(this.feePayerKeyring)

    this.logger.log(
      `Orakl Network Delegator Fee Payer: initialized successfully with address ${this.feePayerKeyring.address}`
    )
  }

  async create(data: SignDto) {
    try {
      data.timestamp = new Date()
      data.from = data.from.toLocaleLowerCase()
      data.to = data.to.toLocaleLowerCase()
      const transaction = await this.prisma.transaction.create({ data })
      const validatedResult = await this.validateTransaction(transaction)
      const signedRawTx = await this.signTxByFeePayer(transaction)
      const signedTransaction = await this.updateTransaction(
        transaction,
        signedRawTx,
        validatedResult
      )
      return signedTransaction
    } catch (e) {
      const msg = `DelegatorError: ${e.name}`
      this.logger.error(msg)
      this.logger.error(e)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }
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

  async updateTransaction(transaction: Transaction, signedRawTx: string, validatedQuery) {
    const data: SignDto = { ...transaction }
    data.succeed = true
    data.signedRawTx = signedRawTx
    data.reporterId = validatedQuery.reporter[0].id
    data.contractId = validatedQuery.id
    data.functionId = validatedQuery.function[0].id

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
    // initialize caver wallet if it is not initialized
    if (!this.feePayerKeyring) {
      await this.initialize({})
    }

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
    const encodedName = tx.input.substring(0, 10)
    const result = await this.prisma.contract.findMany({
      where: {
        address: tx.to,
        reporter: { some: { address: tx.from } },
        function: { some: { encodedName } }
      },
      include: { reporter: { where: { address: tx.from } }, function: { where: { encodedName } } }
    })

    if (result.length == 1 && result[0].function.length == 1 && result[0].reporter.length == 1) {
      return result[0]
    } else if (result.length == 0) {
      throw new DelegatorError(DelegatorErrorCode.NotApprovedTransaction)
    } else {
      throw new DelegatorError(DelegatorErrorCode.UnexpectedResultLength)
    }
  }
}
