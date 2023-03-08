import * as Fs from 'node:fs/promises'
import { Transaction } from '@prisma/client'
import Caver, { AbiItem, SingleKeyring } from 'caver-js'
import { SignDto } from '../dto/sign.dto'
import { dummyFactory } from './dummyFactory'
import { DelegatorError, DelegatorErrorCode } from './errors'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
const caver = new Caver(PROVIDER_URL)
const feePayerKeyring = caver.wallet.keyring.createFromPrivateKey(process.env.SIGNER_PRIVATE_KEY)
caver.wallet.add(feePayerKeyring)
const keyring = caver.wallet.keyring.createFromPrivateKey(process.env.CAVER_PRIVATE_KEY)
caver.wallet.add(keyring)

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}

function encryptMethodName(method: string) {
  return caver.utils.sha3(method).slice(0, 10)
}

async function getSignedTx(txId) {
  // TODO: read Tx it from DB
  const contract = new caver.contract(dummyFactory.abi as AbiItem[], dummyFactory.address)
  const input = contract.methods.increament().encodeABI()
  const rawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
    from: keyring.address,
    to: contract._address,
    input: input,
    gas: 90000
  })
  const newRawTx = caver.transaction.feeDelegatedSmartContractExecution.create({
    from: rawTx.from,
    to: rawTx.to,
    input: rawTx.input,
    gas: rawTx.gas,
    signatures: rawTx.signatures,
    value: rawTx.value,
    chainId: rawTx.chainId,
    gasPrice: rawTx.gasPrice,
    nonce: rawTx.nonce
  })
  await caver.wallet.sign(keyring.address, newRawTx)
  return newRawTx
}

async function signByFeePayerAndExecuteTransaction(rawTx) {
  await caver.wallet.signAsFeePayer(feePayerKeyring.address, rawTx)
  return await caver.rpc.klay.sendRawTransaction(rawTx.getRawTransaction())
}

async function validateTransaction(rawTx) {
  const filePath = './src/sign/whitelist/contractsList.json'
  const contractList = await loadJson(filePath)

  if (!(rawTx.to in contractList)) {
    throw new DelegatorError(DelegatorErrorCode.InvalidContract)
  }
  if (!contractList[rawTx.to].reporters.includes(rawTx.from)) {
    throw new DelegatorError(DelegatorErrorCode.InvalidReporter)
  }
  let isValidMethod = false
  for (const method of contractList[rawTx.to].methods) {
    const encryptedMessage = encryptMethodName(method)
    if (encryptedMessage == rawTx.input) {
      isValidMethod = true
    }
  }
  if (!isValidMethod) {
    throw new DelegatorError(DelegatorErrorCode.InvalidMethod)
  }
}

export async function approveAndSign(input: Transaction) {
  console.log('Input', input)

  const rawTx = await getSignedTx(input.id)
  await validateTransaction(rawTx)

  const receipt = await signByFeePayerAndExecuteTransaction(rawTx)
  console.log('Receipt:', receipt)
}
