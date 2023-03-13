import { Test, TestingModule } from '@nestjs/testing'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import Caver, { AbiItem, Transaction } from 'caver-js'
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
    const tx = caver.transaction.feeDelegatedSmartContractExecution.create({
      from: keyring.address,
      to: contract._address,
      input: input,
      gas: 90000
    })

    await caver.wallet.sign(keyring.address, tx)
    const data: SignDto = {
      from: tx.from,
      to: tx.to,
      input: tx.input,
      gas: tx.gas,
      value: tx.value,
      chainId: tx.chainId,
      gasPrice: tx.gasPrice,
      nonce: tx.nonce,
      v: tx.signatures[0].v,
      r: tx.signatures[0].r,
      s: tx.signatures[0].s,
      rawTx: tx.getRawTransaction()
    }
    const transaction = await service.create(data)
    expect(transaction.signedRawTx)

    const oldNonce = (await caver.rpc.klay.getAccount(keyring.address)).account.nonce
    await caver.rpc.klay.sendRawTransaction(transaction.signedRawTx)

    const newNonce = (await caver.rpc.klay.getAccount(keyring.address)).account.nonce
    expect(oldNonce + 1).toBe(newNonce)
  })
})
