import { HttpException, HttpStatus, Injectable } from '@nestjs/common'
import { Transaction, Prisma } from '@prisma/client'
import { PrismaService } from '../prisma.service'
import { SignDto } from './dto/sign.dto'
import Caver from 'caver-js'
import { SignatureData } from 'caver-js'
import { DelegatorError, DelegatorErrorCode } from './helper/errors'
import { SignTxData } from './helper/types'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
let caver, feePayerKeyring

function encryptMethodName(methodName: string) {
  return caver.klay.abi.encodeFunctionSignature(methodName)
}

async function signTxByFeePayer(input: Transaction) {
  const signature: SignatureData = new caver.wallet.keyring.signatureData([
    input.v,
    input.r,
    input.s
  ])

  const iTx: SignTxData = {
    from: input.from,
    to: input.to,
    input: input.input,
    gas: input.gas,
    signatures: [signature],
    value: input.value,
    chainId: input.chainId,
    gasPrice: input.gasPrice,
    nonce: input.nonce
  }

  const tx = await caver.transaction.feeDelegatedSmartContractExecution.create({ ...iTx })
  await caver.wallet.signAsFeePayer(feePayerKeyring.address, tx)
  return tx.getRawTransaction()
}

function validateTransaction(tx) {
  // FIXME remove simple whiteListing settings and setup db
  const contractList = {
    '0x5b7a8096dd24ceda17f47ae040539dc0566cd1c9': {
      methods: ['increament()', 'decreament()'],
      reporters: ['0x42cbc5b3fb1b7b62fb8bd7c1d475bee35ad3e5f4']
    }
  }

  if (!(tx.to in contractList)) {
    throw new DelegatorError(DelegatorErrorCode.InvalidContract)
  }
  if (!contractList[tx.to].reporters.includes(tx.from)) {
    throw new DelegatorError(DelegatorErrorCode.InvalidReporter)
  }
  const isAllowedMethod = contractList[tx.to].methods.some(
    (method) => encryptMethodName(method) == tx.input
  )
  if (!isAllowedMethod) {
    throw new DelegatorError(DelegatorErrorCode.InvalidMethod)
  }
}

@Injectable()
export class SignService {
  constructor(private prisma: PrismaService) {
    caver = new Caver(PROVIDER_URL)
    feePayerKeyring = caver.wallet.keyring.createFromPrivateKey(process.env.SIGNER_PRIVATE_KEY)
    caver.wallet.add(feePayerKeyring)
  }

  async create(data: SignDto) {
    const transaction = await this.prisma.transaction.create({ data })
    try {
      validateTransaction(transaction)
    } catch (e) {
      const msg = `DelegatorError: ${e.name}`
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }
    const signedRawTx = await signTxByFeePayer(transaction)
    this.updateSignedRawTransaction(transaction.id, signedRawTx)
    return this.findOne({ id: transaction.id })
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
}
