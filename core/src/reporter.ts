import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { buildBullMqConnection, buildQueueName, loadJson, pipe } from './utils.js'
import { reporterRequestQueueName } from './settings.js'
import { IcnError, IcnErrorCode } from './errors.js'

async function reporterJob(wallet) {
  // TODO send data back to Oracle

  async function wrapper(job) {
    console.log(wallet)
  }

  return wrapper
}

async function main() {
  const providerEnv = process.env.PROVIDER
  const mnemonicEnv = process.env.MNEMONIC

  if (mnemonicEnv) {
    if (providerEnv) {
      const provider = new ethers.providers.JsonRpcProvider(providerEnv)
      const wallet = ethers.Wallet.fromMnemonic(mnemonicEnv).connect(provider)
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
