import { Test, TestingModule } from '@nestjs/testing'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import Caver, { AbiItem } from 'caver-js'
import { dummyFactory } from './helper/dummyFactory'
import { SignDto } from './dto/sign.dto'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
const caver = new Caver(PROVIDER_URL)
const keyring = caver.wallet.keyring.createFromPrivateKey(process.env.CAVER_PRIVATE_KEY)
caver.wallet.add(keyring)

describe('SignService', () => {
  let service: SignService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [SignService, PrismaService]
    }).compile()

    service = module.get<SignService>(SignService)
  })

  it('SignedRawTx should not be empty/null', async () => {
    const contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
    const input = contract.methods.increament().encodeABI()
    const rawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
      from: keyring.address,
      to: contract._address,
      input: input,
      gas: 90000
    })

    await caver.wallet.sign(keyring.address, rawTx)
    const data: SignDto = {
      from: rawTx.from,
      to: rawTx.to,
      input: rawTx.input,
      gas: rawTx.gas,
      value: rawTx.value,
      chainId: rawTx.chainId,
      gasPrice: rawTx.gasPrice,
      nonce: rawTx.nonce,
      v: rawTx.signatures[0].v,
      r: rawTx.signatures[0].r,
      s: rawTx.signatures[0].s,
      rawTx: rawTx.getRawTransaction()
    }
    const transactionId = await service.create(data)
    const transaction = await service.findOne({ id: transactionId })
    console.log('TransactionId', transactionId)
    console.log('Transaction', transaction.signedRawTx)
  })
})
