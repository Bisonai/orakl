import { HttpException, HttpStatus, Injectable, Logger } from '@nestjs/common'
import Caver, { SignatureData } from 'caver-js'
import { PrismaService } from '../prisma.service'
import { SignDto } from './dto/sign.dto'
import { DelegatorError, DelegatorErrorCode } from './errors'

interface TransactionType {
  timestamp: Date
  from: string
  to: string
  input: string
  gas: string
  value: string
  chainId: string
  gasPrice: string
  nonce: string
  v: string
  r: string
  s: string
  rawTx: string
  signedRawTx: string
  succeed: boolean
  functionId: bigint
  contractId: bigint
  reporterId: bigint
}

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
      const transaction = getTransactionType(data)
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

  async updateTransaction(transaction: TransactionType, signedRawTx: string, validatedQuery) {
    const data: SignDto = { ...transaction }
    data.succeed = true
    data.signedRawTx = signedRawTx
    data.reporterId = validatedQuery.reporter[0].id
    data.contractId = validatedQuery.id
    data.functionId = validatedQuery.function[0].id

    return data
  }

  async signTxByFeePayer(input: TransactionType) {
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

function getTransactionType(data: SignDto): TransactionType {
  return {
    timestamp: new Date(data.timestamp),
    from: data.from.toLocaleLowerCase(),
    to: data.to.toLocaleLowerCase(),
    input: data.input,
    gas: data.gas,
    value: data.value,
    chainId: data.chainId,
    gasPrice: data.gasPrice,
    nonce: data.nonce,
    v: data.v,
    r: data.r,
    s: data.s,
    rawTx: data.rawTx,
    signedRawTx: data.signedRawTx,
    succeed: data.succeed,
    functionId: data.functionId,
    contractId: data.contractId,
    reporterId: data.reporterId
  }
}
