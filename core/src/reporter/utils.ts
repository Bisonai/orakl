import { ethers } from 'ethers'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { PROVIDER_URL as PROVIDER_ENV, PRIVATE_KEY as PRIVATE_KEY_ENV } from '../settings'
import { add0x } from '../utils'
import { NonceManager } from '@ethersproject/experimental'

const FILE_NAME = import.meta.url

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

export function loadWalletParameters() {
  if (!PRIVATE_KEY_ENV) {
    throw new OraklError(OraklErrorCode.MissingMnemonic)
  }

  if (!PROVIDER_ENV) {
    throw new OraklError(OraklErrorCode.MissingJsonRpcProvider)
  }

  return { privateKey: PRIVATE_KEY_ENV, providerUrl: PROVIDER_ENV }
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
