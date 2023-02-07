import { ethers } from 'ethers'
import { Logger } from 'pino'
import { IcnError, IcnErrorCode } from '../errors'
import { PROVIDER_URL as PROVIDER_ENV, PRIVATE_KEY as PRIVATE_KEY_ENV } from '../settings'
import { add0x } from '../utils'

const FILE_NAME = import.meta.url

export async function buildWallet({
  privateKey,
  providerUrl,
  testConnection
}: {
  privateKey: string
  providerUrl: string
  testConnection?: boolean
}) {
  const provider = new ethers.providers.JsonRpcProvider(providerUrl)
  const wallet = new ethers.Wallet(privateKey, provider)

  if (testConnection) {
    try {
      await wallet.getTransactionCount()
    } catch (e) {
      if (e.code == 'NETWORK_ERROR') {
        throw new IcnError(IcnErrorCode.ProviderNetworkError, 'ProviderNetworkError', e.reason)
      }
    }
  }

  return wallet
}

export function loadWalletParameters() {
  if (!PRIVATE_KEY_ENV) {
    throw new IcnError(IcnErrorCode.MissingMnemonic)
  }

  if (!PROVIDER_ENV) {
    throw new IcnError(IcnErrorCode.MissingJsonRpcProvider)
  }

  return { privateKey: PRIVATE_KEY_ENV, providerUrl: PROVIDER_ENV }
}

export async function sendTransaction({
  wallet,
  to,
  payload,
  gasLimit,
  value,
  _logger
}: {
  wallet
  to: string
  payload?: string
  gasLimit?: number | string
  value?: number | string | ethers.BigNumber
  _logger?: Logger
}) {
  const logger = _logger?.child({ name: 'sendTransaction', file: FILE_NAME })

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
  logger?.debug(tx, 'tx')

  try {
    const txReceipt = await (await wallet.sendTransaction(tx)).wait()
    logger?.debug(txReceipt, 'txReceipt')
  } catch (e) {
    logger?.debug(e, 'e')

    if (e.reason == 'invalid address') {
      throw new IcnError(IcnErrorCode.TxInvalidAddress, 'TxInvalidAddress', e.value)
    } else if (e.reason == 'processing response error') {
      throw new IcnError(
        IcnErrorCode.TxProcessingResponseError,
        'TxProcessingResponseError',
        e.value
      )
    } else if (e.code == 'UNPREDICTABLE_GAS_LIMIT') {
      throw new IcnError(IcnErrorCode.TxCannotEstimateGasError, 'TxCannotEstimateGasError', e.value)
    }
  }
}
