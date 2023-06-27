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
import { PrismaClient } from '@prisma/client'

const caver = new Caver(process.env.PROVIDER_URL)
const keyring = caver.wallet.keyring.createFromPrivateKey(process.env.TEST_DELEGATOR_REPORTER_PK)
caver.wallet.add(keyring)

describe('SignService', () => {
  let service: SignService
  let organizationService: OrganizationService
  let contractService: ContractService
  let functionService: FunctionService
  let reporterService: ReporterService
  let transactionData: SignDto, contract
  let prisma

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
    await service.initialize({ feePayerPrivateKey: process.env.DELEGATOR_FEEPAYER_PK })
    organizationService = module.get<OrganizationService>(OrganizationService)
    contractService = module.get<ContractService>(ContractService)
    functionService = module.get<FunctionService>(FunctionService)
    reporterService = module.get<ReporterService>(ReporterService)
    prisma = module.get<PrismaClient>(PrismaService)

    contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
    const input = contract.methods.increment().encodeABI()
    const tx = caver.transaction.feeDelegatedSmartContractExecution.create({
      from: keyring.address,
      to: contract._address,
      input: input,
      gas: 90000
    })

    await caver.wallet.sign(keyring.address, tx)
    transactionData = {
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
  })

  afterEach(async () => {
    jest.resetModules()
    await prisma.$disconnect()
  })

  it('Should validateTransaction and execute transaction', async () => {
    // Setup Organization
    const organizationName = 'BisonAI'
    const org = await organizationService.create({ name: organizationName })
    expect(org.name).toBe(organizationName)

    // Setup reporter
    const rep = await reporterService.create({
      address: transactionData.from,
      organizationId: org.id
    })
    expect(rep.address).toBe(transactionData.from)

    // Setup Contract
    const con = await contractService.create({ address: transactionData.to })
    expect(con.address).toBe(transactionData.to)

    // Connect Contract to Reporter
    await contractService.connectReporter({ contractId: con.id, reporterId: rep.id })

    // Setup functionName
    const functionMethod = 'increment()'
    const fun = await functionService.create({ name: functionMethod, contractId: con.id })
    expect(fun.name).toBe(functionMethod)

    const transaction = await service.create(transactionData)
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

  it('Should validateTransaction and execute transaction', async () => {
    // Setup Organization & Reporter 1
    const organizationName1 = 'Company'
    const org1 = await organizationService.create({ name: organizationName1 })
    expect(org1.name).toBe(organizationName1)

    const dummyAddress = '0x000000000000000'
    const rep1 = await reporterService.create({
      address: dummyAddress,
      organizationId: org1.id
    })
    expect(rep1.address).toBe(dummyAddress)

    // Setup Organization & Reporter 2
    const organizationName2 = 'Bisonai'
    const org2 = await organizationService.create({ name: organizationName2 })
    expect(org2.name).toBe(organizationName2)

    const rep2 = await reporterService.create({
      address: transactionData.from,
      organizationId: org2.id
    })
    expect(rep2.address).toBe(transactionData.from)

    // Setup Contract
    const con = await contractService.create({ address: transactionData.to })
    expect(con.address).toBe(transactionData.to)

    // Connect Contract to Reporters
    await contractService.connectReporter({ contractId: con.id, reporterId: rep1.id })
    await contractService.connectReporter({ contractId: con.id, reporterId: rep2.id })

    // Setup 2 functionName
    const functionMethod1 = 'dummy()'
    const fun1 = await functionService.create({ name: functionMethod1, contractId: con.id })
    expect(fun1.name).toBe(functionMethod1)

    const functionMethod2 = 'increment()'
    const fun2 = await functionService.create({ name: functionMethod2, contractId: con.id })
    expect(fun2.name).toBe(functionMethod2)

    const transaction = await service.create(transactionData)

    expect(transaction.signedRawTx)
    expect(transaction.reporterId).toBe(rep2.id)
    expect(transaction.functionId).toBe(fun2.id)

    const oldCounter = await contract.methods.COUNTER().call()
    await caver.rpc.klay.sendRawTransaction(transaction.signedRawTx)
    const newCounter = await contract.methods.COUNTER().call()
    expect(Number(oldCounter) + 1).toBe(Number(newCounter))

    // cleanup
    await functionService.remove({ id: fun1.id })
    await functionService.remove({ id: fun2.id })
    await reporterService.remove({ id: rep1.id })
    await reporterService.remove({ id: rep2.id })
    await organizationService.remove({ id: org1.id })
    await organizationService.remove({ id: org2.id })
    await contractService.remove({ id: con.id })
  })

  it('Should fail to validateTransaction, when reporter not connected to contract', async () => {
    // Setup Organization
    const organizationName = 'BisonAI'
    const org = await organizationService.create({ name: organizationName })
    expect(org.name).toBe(organizationName)

    // Setup Contract
    const con = await contractService.create({ address: transactionData.to })
    expect(con.address).toBe(transactionData.to)

    // Setup functionName
    const functionMethod = 'increment()'
    const fun = await functionService.create({ name: functionMethod, contractId: con.id })
    expect(fun.name).toBe(functionMethod)

    // expect to fail without no reporter
    await expect(async () => {
      await service.create(transactionData)
    }).rejects.toThrow()

    // Setup reporter
    const rep = await reporterService.create({
      address: transactionData.from,
      organizationId: org.id
    })

    // expect to fail without reporter is connect to contract
    await expect(async () => {
      await service.create(transactionData)
    }).rejects.toThrow()

    // Connect Contract to Reporter
    await contractService.connectReporter({ contractId: con.id, reporterId: rep.id })

    // expect to transaction approved and signed
    const transaction = await service.create(transactionData)
    expect(transaction.signedRawTx)

    // cleanup
    await reporterService.remove({ id: rep.id })
    await functionService.remove({ id: fun.id })
    await organizationService.remove({ id: org.id })
    await contractService.remove({ id: con.id })
  })

  it('Should fail to validateTransaction, when incorrect function method allowed', async () => {
    // Setup Organization
    const organizationName = 'BisonAI'
    const org = await organizationService.create({ name: organizationName })
    expect(org.name).toBe(organizationName)

    // Setup reporter
    const rep = await reporterService.create({
      address: transactionData.from,
      organizationId: org.id
    })
    expect(rep.address).toBe(transactionData.from)
    // Setup Contract
    const con = await contractService.create({ address: transactionData.to })
    expect(con.address).toBe(transactionData.to)

    // Connect Contract to Reporter
    await contractService.connectReporter({ contractId: con.id, reporterId: rep.id })

    // expect to fail without no function
    await expect(async () => {
      await service.create(transactionData)
    }).rejects.toThrow()

    // Setup wrong functionName
    const incorrectFunction = 'decrement()'
    const fun = await functionService.create({ name: incorrectFunction, contractId: con.id })
    expect(fun.name).toBe(incorrectFunction)

    // expect to fail with wrong function name
    await expect(async () => {
      await service.create(transactionData)
    }).rejects.toThrow()

    // // cleanup
    await functionService.remove({ id: fun.id })
    await reporterService.remove({ id: rep.id })
    await organizationService.remove({ id: org.id })
    await contractService.remove({ id: con.id })
  })
})
