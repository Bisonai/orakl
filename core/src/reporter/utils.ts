import axios from 'axios'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { NonceManager } from '@ethersproject/experimental'
import Caver from 'caver-js'
import { OraklError, OraklErrorCode } from '../errors'
import { ORAKL_NETWORK_DELEGATOR_URL } from '../settings'
import { add0x, buildUrl } from '../utils'
import { ITransactionData } from '../types'

const FILE_NAME = import.meta.url

export class CaverWallet {
  caver: Caver
  address: string

  constructor(privateKey: string, providerUrl: string) {
    this.caver = new Caver(providerUrl)
    const keyring = this.caver.wallet.keyring.createFromPrivateKey(privateKey)
    this.caver.wallet.add(keyring)
    this.address = keyring.address
    // FIXME utilize nonce manager
  }
}

export function buildWallet({
  privateKey,
  providerUrl
}: {
  privateKey: string
  providerUrl: string
}) {
  const provider = new ethers.providers.JsonRpcProvider(providerUrl)
  const basicWallet = new ethers.Wallet(privateKey, provider)
  const wallet = new NonceManager(basicWallet)
  return wallet
}

export function buildCaverWallet({
  privateKey,
  providerUrl
}: {
  privateKey: string
  providerUrl: string
}) {
  const caverWallet = new CaverWallet(privateKey, providerUrl)
  return caverWallet
}

export async function testConnection(wallet: ethers.Wallet) {
  try {
    await wallet.getTransactionCount()
  } catch (e) {
    if (e.code == 'NETWORK_ERROR') {
      throw new OraklError(OraklErrorCode.ProviderNetworkError, 'ProviderNetworkError', e.reason)
    } else {
      throw e
    }
  }
}

export async function sendTransaction({
  wallet,
  to,
  payload,
  gasLimit,
  value,
  logger
}: {
  wallet
  to: string
  payload?: string
  gasLimit?: number | string
  value?: number | string | ethers.BigNumber
  logger: Logger
}) {
  const _logger = logger.child({ name: 'sendTransaction', file: FILE_NAME })

  if (payload) {
    payload = add0x(payload)
  }

  const tx = {
    from: wallet.address,
    to: to,
    data: payload || '0x00',
    value: value || '0x00'
  }

  if (gasLimit) {
    tx['gasLimit'] = gasLimit
  }
  _logger.debug(tx, 'tx')

  try {
    const txReceipt = await (await wallet.sendTransaction(tx)).wait(1)
    _logger.debug(txReceipt, 'txReceipt')
    if (txReceipt === null) {
      throw new OraklError(OraklErrorCode.TxNotMined)
    }
  } catch (e) {
    _logger.debug(e, 'e')

    let msg
    let error
    if (e.reason == 'invalid address') {
      msg = `TxInvalidAddress ${e.value}`
      error = new OraklError(OraklErrorCode.TxInvalidAddress, msg, e.value)
    } else if (e.reason == 'processing response error') {
      msg = `TxProcessingResponseError ${e.value}`
      error = new OraklError(OraklErrorCode.TxProcessingResponseError, msg, e.value)
    } else if (e.reason == 'missing response') {
      msg = 'TxMissingResponseError'
      error = new OraklError(OraklErrorCode.TxMissingResponseError, msg)
    } else if (e.reason == 'transaction failed') {
      msg = 'TxTransactionFailed'
      error = new OraklError(OraklErrorCode.TxTransactionFailed, msg)
    } else if (e.reason == 'insufficient funds for intrinsic transaction cost') {
      msg = 'TxInsufficientFunds'
      error = new OraklError(OraklErrorCode.TxProcessingResponseError, msg)
    } else if (e.code == 'UNPREDICTABLE_GAS_LIMIT') {
      msg = 'TxCannotEstimateGasError'
      error = new OraklError(OraklErrorCode.TxCannotEstimateGasError, msg, e.value)
    } else {
      error = e
    }

    _logger.error(msg)
    throw error
  }
}

export async function sendTransactionDelegatedFee({
  wallet,
  to,
  payload,
  gasLimit,
  value,
  logger
}: {
  wallet: CaverWallet
  to: string
  payload?: string
  gasLimit?: number | string
  value?: number | string
  logger: Logger
}) {
  const _logger = logger.child({ name: 'sendTransactionDelegatedFee', file: FILE_NAME })

  const txParams = {
    from: wallet.address,
    to,
    input: payload,
    gas: gasLimit,
    value: value || '0x00'
  }
  const tx = wallet.caver.transaction.feeDelegatedSmartContractExecution.create(txParams)
  await wallet.caver.wallet.sign(wallet.address, tx)

  const transactionData: ITransactionData = {
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
  _logger.debug(transactionData)

  const endpoint = buildUrl(ORAKL_NETWORK_DELEGATOR_URL, `sign`)

  let response
  try {
    response = (
      await axios.post(endpoint, {
        ...transactionData
      })
    )?.data
  } catch (e) {
    throw new OraklError(OraklErrorCode.DelegatorServerIssue)
  }

  try {
    if (response?.signedRawTx) {
      const txReceipt = await wallet.caver.rpc.klay.sendRawTransaction(response.signedRawTx)
      _logger.debug(txReceipt, 'txReceipt')
      return txReceipt
    } else {
      throw new OraklError(OraklErrorCode.MissingSignedRawTx)
    }
  } catch (e) {
    _logger.error(e, 'e')
    throw e
  }
}

export async function sendTransactionCaver({
  wallet,
  to,
  payload,
  gasLimit,
  logger,
  value
}: {
  wallet: CaverWallet
  to: string
  payload: string
  gasLimit: number | string
  logger: Logger
  value?: number | string
}) {
  const txParams = {
    from: wallet.address,
    to,
    input: payload,
    gasLimit,
    value: value || '0x00'
  }

  try {
    const tx = wallet.caver.transaction.smartContractExecution.create(txParams)
    await wallet.caver.wallet.sign(wallet.address, tx)
    const txReceipt = await wallet.caver.rpc.klay.sendRawTransaction(tx.getRawTransaction())
  } catch (e) {
    throw new OraklError(OraklErrorCode.CaverTxTransactionFailed)
  }
}
