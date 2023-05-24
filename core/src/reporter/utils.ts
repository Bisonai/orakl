import axios from 'axios'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { NonceManager } from '@ethersproject/experimental'
import Caver, { AbiItem } from 'caver-js'
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

    if (e.reason == 'invalid address') {
      const msg = `TxInvalidAddress ${e.value}`
      _logger.error(msg)

      throw new OraklError(OraklErrorCode.TxInvalidAddress, 'TxInvalidAddress', e.value)
    } else if (e.reason == 'processing response error') {
      const msg = `TxProcessingResponseError ${e.value}`
      _logger.error(msg)

      throw new OraklError(
        OraklErrorCode.TxProcessingResponseError,
        'TxProcessingResponseError',
        e.value
      )
    } else if (e.reason == 'missing response') {
      const msg = 'TxMissingResponseError'
      _logger.error(msg)

      throw new OraklError(OraklErrorCode.TxMissingResponseError, 'TxMissingResponseError')
    } else if (e.reason == 'transaction failed') {
      const msg = 'TxTransactionFailed'
      _logger.error(msg)

      throw new OraklError(OraklErrorCode.TxTransactionFailed, 'TxTransactionFailed')
    } else if (e.code == 'UNPREDICTABLE_GAS_LIMIT') {
      const msg = 'TxCannotEstimateGasError'
      _logger.error(msg)

      throw new OraklError(
        OraklErrorCode.TxCannotEstimateGasError,
        'TxCannotEstimateGasError',
        e.value
      )
    } else {
      throw e
    }
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
  value?: number | string | ethers.BigNumber
  logger: Logger
}) {
  const _logger = logger.child({ name: 'sendTransactionDelegatedFee', file: FILE_NAME })

  const tx = wallet.caver.transaction.feeDelegatedSmartContractExecution.create({
    from: wallet.address,
    to,
    input: payload,
    gas: gasLimit
  })
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

  try {
    const result = (
      await axios.post(endpoint, {
        ...transactionData
      })
    )?.data

    if (result?.signedRawTx) {
      const txReceipt = await wallet.caver.rpc.klay.sendRawTransaction(result.signedRawTx)
      _logger.debug(txReceipt, 'txReceipt')
    } else {
      throw new OraklError(OraklErrorCode.MissingSignedRawTx)
    }
  } catch (e) {
    _logger.error(e, 'e')
    throw e
  }
}
