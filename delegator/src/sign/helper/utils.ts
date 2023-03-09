import * as Fs from 'node:fs/promises'
import { Transaction } from '@prisma/client'
import Caver from 'caver-js'
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

async function signByFeePayerAndExecuteTransaction(input: Transaction) {
  // TODO: decode Tx from rawTx
  // let tx = caver.transaction.decode(input.rawTx)
  // console.log('rawTx:', tx)

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
  await validateTransaction(input)
  return await signByFeePayerAndExecuteTransaction(input)
}
