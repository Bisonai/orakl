import { Transaction } from '@prisma/client'
import Caver from 'caver-js'
import { DelegatorError, DelegatorErrorCode } from './errors'
import { SignatureData } from 'caver-js'
import { SignTxData } from './types'

const PROVIDER_URL = 'https://api.baobab.klaytn.net:8651'
const caver = new Caver(PROVIDER_URL)
const feePayerKeyring = caver.wallet.keyring.createFromPrivateKey(process.env.SIGNER_PRIVATE_KEY)
caver.wallet.add(feePayerKeyring)

function encryptMethodName(method: string) {
  return caver.klay.abi.encodeFunctionSignature(method)
}

export async function signTxByFeePayer(input: Transaction) {
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

export function validateTransaction(rawTx) {
  // FIXME remove simple whiteListing settings and setup db
  const contractList = {
    '0x5b7a8096dd24ceda17f47ae040539dc0566cd1c9': {
      methods: ['increament()', 'decreament()'],
      reporters: ['0x42cbc5b3fb1b7b62fb8bd7c1d475bee35ad3e5f4']
    }
  }

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
