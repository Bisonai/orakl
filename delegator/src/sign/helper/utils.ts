import * as Fs from 'node:fs/promises'
import { Transaction } from '@prisma/client'
import Caver, { AbiItem, SingleKeyring } from 'caver-js'
import { SignDto } from '../dto/sign.dto'
import { dummyFactory } from './dummyFactory'
import { DelegatorError, DelegatorErrorCode } from './errors'
import { SignatureData } from 'caver-js'
import { SignTxData } from './types'

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
  return caver.klay.abi.encodeFunctionSignature(method)
}

async function getSignedTx(rawTx) {
  const signature: SignatureData = new caver.wallet.keyring.signatureData([
    rawTx.v,
    rawTx.r,
    rawTx.s
  ])

  const iTx: SignTxData = {
    from: rawTx.from,
    to: rawTx.to,
    input: rawTx.input,
    gas: rawTx.gas,
    signatures: [signature],
    value: rawTx.value,
    chainId: rawTx.chainId,
    gasPrice: rawTx.gasPrice,
    nonce: rawTx.nonce
  }

  return await caver.transaction.feeDelegatedSmartContractExecution.create({ ...iTx })
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
  const rawTx = await getSignedTx(input)
  await validateTransaction(rawTx)

  const receipt = await signByFeePayerAndExecuteTransaction(rawTx)
  console.log('Receipt:', receipt)
}
