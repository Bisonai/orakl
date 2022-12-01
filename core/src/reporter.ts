import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { buildBullMqConnection, buildQueueName, loadJson, pipe } from './utils.js'
import { reporterRequestQueueName } from './settings.js'
import { IcnError, IcnErrorCode } from './errors.js'

function pad32Bytes(data) {
  let s = String(data)
  while (s.length < (64 || 2)) {
    s = '0' + s
  }
  return s
}

// function makeFunctionSelector(fnSignature) {
//   return ethers.utils.id(fnSignature).slice(2).slice(0, 8)
// }

async function reporterJob(wallet) {
  // TODO send data back to Oracle

  async function wrapper(job) {
    // console.log(wallet)
    console.log(job.data)

    try {
      // const fnSelector = makeFunctionSelector('fulfill(bytes32,int256)')
      const requestIdParam = pad32Bytes(job.data.requestId.slice(2))
      const responseData = Math.floor(job.data.data) // FIXME
      const responseParam = pad32Bytes(ethers.utils.hexlify(responseData).slice(2))
      const payload = job.data.callbackFunctionId.slice(2) + requestIdParam + responseParam

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
