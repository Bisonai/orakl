import { ethers } from 'ethers'
import { IcnError, IcnErrorCode } from '../errors'
import { PROVIDER_URL as PROVIDER_ENV, PRIVATE_KEY as PRIVATE_KEY_ENV } from '../settings'
import { add0x } from '../utils'

export function buildWallet() {
  try {
    const { PRIVATE_KEY, PROVIDER } = checkParameters()
    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
    const wallet = new ethers.Wallet(PRIVATE_KEY, provider)
    return wallet
  } catch (e) {
    console.error(e)
  }
}

function checkParameters() {
  if (!PRIVATE_KEY_ENV) {
    throw new IcnError(IcnErrorCode.MissingMnemonic)
  }

  if (!PROVIDER_ENV) {
    throw new IcnError(IcnErrorCode.MissingJsonRpcProvider)
  }

  return { PRIVATE_KEY: PRIVATE_KEY_ENV, PROVIDER: PROVIDER_ENV }
}

export async function sendTransaction(wallet, to, payload, gasLimit?, value?) {
  const tx = {
    from: wallet.address,
    to: to,
    data: add0x(payload),
    value: value || '0x00'
  }

  if (gasLimit) {
    tx['gasLimit'] = gasLimit
  }

  console.debug('sendTransaction:tx')
  console.debug(tx)

  const txReceipt = await wallet.sendTransaction(tx)
  console.debug('sendTransaction:txReceipt')
  console.debug(txReceipt)
}
