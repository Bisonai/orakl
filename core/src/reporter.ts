import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { VRFCoordinator__factory } from '@bisonai/icn-contracts'
import { buildBullMqConnection, buildQueueName, loadJson, pipe, remove0x } from './utils'
import { REPORTER_REQUEST_QUEUE_NAME, REPORTER_VRF_QUEUE_NAME } from './settings'
import { IcnError, IcnErrorCode } from './errors'
import { PROVIDER, MNEMONIC, PRIVATE_KEY } from './load-parameters'
import { pad32Bytes, add0x } from './utils'

async function sendTransaction(wallet, _from, to, payload, gasLimit?, value?) {
  const tx = {
    from: _from,
    to: to,
    data: add0x(payload),
    gasLimit: gasLimit || '0x34710', // FIXME
    value: value || '0x00'
  }
  console.debug('sendTransaction:tx')
  console.debug(tx)

  const txReceipt = await wallet.sendTransaction(tx)
  console.debug(`sendTransaction:txReceipt ${txReceipt}`)
}

function vrfJob(wallet) {
  async function wrapper(job) {
    const data = job.data
    console.log('vrfJob', data)

    try {
      const requestCommitment = [
        data.blockNum,
        data.subId,
        data.callbackGasLimit,
        data.numWords,
        data.sender
      ]
      console.log('requestCommitment', requestCommitment)

      const proof = [data.pk, data.proof, data.preSeed, data.uPoint, data.vComponents]
      console.log('proof', proof)

      let iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
      const payload = iface.encodeFunctionData('fulfillRandomWords', [proof, requestCommitment])

      console.log('data.callbackAddress', data.callbackAddress)
      const gasLimit = 3_000_000 // FIXME
      await sendTransaction(wallet, wallet.address, data.callbackAddress, payload, gasLimit)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function reporterJob(wallet) {
  // TODO send data back to Oracle

  async function wrapper(job) {
    console.log(job.data)

    try {
      const requestIdParam = pad32Bytes(job.data.requestId)
      const responseData = Math.floor(job.data.data) // FIXME change response based on jobId
      const responseParam = pad32Bytes(ethers.utils.hexlify(responseData))
      const payload = remove0x(job.data.callbackFunctionId) + requestIdParam + responseParam

      const tx = {
        from: wallet.address,
        to: job.data.callbackAddress,
        data: '0x' + payload,
        gasLimit: '0x34710', // FIXME
        value: '0x00' // FIXME
      }
      console.log(tx)

      const txReceipt = await wallet.sendTransaction(tx)
      console.log(txReceipt)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

async function main() {
  if (PRIVATE_KEY) {
    if (PROVIDER) {
      const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
      // const wallet = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
      const wallet = new ethers.Wallet(PRIVATE_KEY, provider)
      // TODO if job not finished, return job in queue

      new Worker(REPORTER_REQUEST_QUEUE_NAME, await reporterJob(wallet), buildBullMqConnection())

      new Worker(REPORTER_VRF_QUEUE_NAME, await vrfJob(wallet), buildBullMqConnection())
    } else {
      throw new IcnError(IcnErrorCode.MissingJsonRpcProvider)
    }
  } else {
    throw new IcnError(IcnErrorCode.MissingMnemonic)
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
