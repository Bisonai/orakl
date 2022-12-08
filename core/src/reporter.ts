import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { buildBullMqConnection, buildQueueName, loadJson, pipe, remove0x } from './utils'
import { REPORTER_REQUEST_QUEUE_NAME, REPORTER_VRF_QUEUE_NAME } from './settings'
import { IcnError, IcnErrorCode } from './errors'
import { PROVIDER, MNEMONIC, PRIVATE_KEY } from './load-parameters'

function pad32Bytes(data) {
  data = remove0x(data)
  let s = String(data)
  while (s.length < (64 || 2)) {
    s = '0' + s
  }
  return s
}

function vrfJob(wallet) {
  async function wrapper(job) {
    const data = job.data
    console.log('vrfJob', job.data)

    console.log(`requestId ${data.requestId}`)
    console.log(`alpha ${data.alpha}`)
    console.log(`callbackGasLimit ${data.callbackGasLimit}`)
    console.log(`sender ${data.sender}`)
    console.log(`proof ${data.proof}`)
    console.log(`beta ${data.beta}`)

    try {
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
