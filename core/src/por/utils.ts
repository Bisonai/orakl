import { ethers } from 'ethers'
import { RPC_URL_TIMEOUT } from '../settings'

export async function checkRpcUrl(url: string) {
  try {
    const provider = new ethers.providers.JsonRpcProvider(url)
    const blockNumberPromise = provider.getBlockNumber()
    const result = await callWithTimeout(blockNumberPromise, RPC_URL_TIMEOUT)

    if (result instanceof Error && result.message === 'Timeout') {
      console.error(`failed to connect rpc url due to timeout: ${url}`)
      return false
    } else {
      console.info(`json rpc is alive: ${url}`)
      return true
    }
  } catch (error) {
    console.error(`Error connecting to URL ${url}: ${error.message}`)
    return false
  }
}

export const callWithTimeout = (promise, timeout) =>
  Promise.race([
    promise,
    new Promise((_, reject) => setTimeout(() => reject(new Error('Timeout')), timeout))
  ])
