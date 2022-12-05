import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { buildBullMqConnection, buildQueueName, loadJson, pipe, remove0x } from './utils.js'
import { reporterRequestQueueName } from './settings.js'
import { IcnError, IcnErrorCode } from './errors.js'

function pad32Bytes(data) {
  data = remove0x(data)
  let s = String(data)
  while (s.length < (64 || 2)) {
    s = '0' + s
  }
  return s
}

async function reporterJob(wallet) {
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
      console.log(e)
    }
  }

  return wrapper
}

async function main() {
  const providerEnv = process.env.PROVIDER
  // FIXME allow either private key or mnemonic
  // const mnemonicEnv = process.env.MNEMONIC
  const privateKeyEnv = process.env.PRIVATE_KEY

  if (privateKeyEnv) {
    if (providerEnv) {
      const provider = new ethers.providers.JsonRpcProvider(providerEnv)
      // const wallet = ethers.Wallet.fromMnemonic(mnemonicEnv).connect(provider)
      const wallet = new ethers.Wallet(privateKeyEnv, provider)
      // TODO if job not finished, return job in queue
      const worker = new Worker(
        reporterRequestQueueName,
        await reporterJob(wallet),
        buildBullMqConnection()
      )
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
