import { Test, TestingModule } from '@nestjs/testing'
import { SignService } from './sign.service'
import { PrismaService } from '../prisma.service'
import Caver, { AbiItem } from 'caver-js'
import { dummyFactory } from './dummyFactory'
import { SignDto } from './dto/sign.dto'
import { OrganizationService } from '../organization/organization.service'
import { ContractService } from '../contract/contract.service'
import { FunctionService } from '../function/function.service'
import { ReporterService } from '../reporter/reporter.service'

const caver = new Caver(process.env.PROVIDER_URL)
const keyring = caver.wallet.keyring.createFromPrivateKey(process.env.DELEGATOR_REPORTER_PK)
caver.wallet.add(keyring)

describe('SignService', () => {
  let service: SignService
  let organizationService: OrganizationService
  let contractService: ContractService
  let functionService: FunctionService
  let reporterService: ReporterService

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        SignService,
        PrismaService,
        OrganizationService,
        ContractService,
        FunctionService,
        ReporterService
      ]
    }).compile()
    service = module.get<SignService>(SignService)
    organizationService = module.get<OrganizationService>(OrganizationService)
    contractService = module.get<ContractService>(ContractService)
    functionService = module.get<FunctionService>(FunctionService)
    reporterService = module.get<ReporterService>(ReporterService)
  })

  it('SignedRawTx should not be empty/null', async () => {
    const contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
    const input = contract.methods.increment().encodeABI()
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

    // Setup Organization
    const organizationName = 'BisonAI'
    const org = await organizationService.create({ name: organizationName })
    expect(org.name).toBe(organizationName)

    // Setup reporter
    const rep = await reporterService.create({
      address: tx.from,
      organizationId: org.id
    })
    expect(rep.address).toBe(tx.from)

    // Setup Contract
    const con = await contractService.create({
      address: tx.to
    })

    // Connect Contract to Reporter
    await contractService.connectReporter({
      contractId: con.id,
      reporterId: rep.id
    })

    expect(con.address).toBe(tx.to)

    // Setup functionName
    const functionMethod = 'increment()'
    const fun = await functionService.create({
      name: functionMethod,
      contractId: con.id
    })
    expect(fun.name).toBe(functionMethod)

    const transaction = await service.create(data)
    expect(transaction.signedRawTx)

    const oldCounter = await contract.methods.COUNTER().call()
    await caver.rpc.klay.sendRawTransaction(transaction.signedRawTx)
    const newCounter = await contract.methods.COUNTER().call()
    expect(Number(oldCounter) + 1).toBe(Number(newCounter))

    // cleanup
    await functionService.remove({ id: fun.id })
    await reporterService.remove({ id: rep.id })
    await organizationService.remove({ id: org.id })
    await contractService.remove({ id: con.id })
  })
})
