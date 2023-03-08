import { Test, TestingModule } from '@nestjs/testing'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import Caver, { AbiItem, SingleKeyring } from 'caver-js'
import { dummyFactory } from './helper/dummyFactory'
import { SignatureData } from 'caver-js'

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

  it('should be defined', () => {
    expect(service).toBeDefined()
  })

  it('should be defined', async () => {
    const contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
    const input = contract.methods.increament().encodeABI()
    const rawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
      from: keyring.address,
      to: contract._address,
      input: input,
      gas: 90000
    })
    await caver.wallet.sign(keyring.address, rawTx)
    const data = {
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
      s: rawTx.signatures[0].s
    }
    console.log(data)

    await service.create(data)
  })

  it('Show Result', async () => {
    const result = await service.findOne({ id: 1 })
    console.log('Result', result)
  })
})
