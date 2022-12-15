import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { ICNOracle__factory, VRFCoordinator__factory } from '@bisonai/icn-contracts'
import { sendTransaction } from './utils'
import { REPORTER_ANY_API_QUEUE_NAME, REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from './settings'
import { IAnyApiWorkerReporter, IVrfWorkerReporter, RequestCommitment, Proof } from './types'
import { IcnError, IcnErrorCode } from './errors'
import {
  PROVIDER as PROVIDER_ENV,
  PRIVATE_KEY as PRIVATE_KEY_ENV,
  MNEMONIC
} from './load-parameters'
import { pad32Bytes } from './utils'

async function main() {
  try {
    const { PRIVATE_KEY, PROVIDER } = checkParameters()

    const provider = new ethers.providers.JsonRpcProvider(PROVIDER)
    // const wallet = ethers.Wallet.fromMnemonic(MNEMONIC).connect(provider)
    const wallet = new ethers.Wallet(PRIVATE_KEY, provider)
    // TODO if job not finished, return job in queue

    new Worker(REPORTER_ANY_API_QUEUE_NAME, await anyApiJob(wallet), BULLMQ_CONNECTION)
    new Worker(REPORTER_VRF_QUEUE_NAME, await vrfJob(wallet), BULLMQ_CONNECTION)
    // TODO Predefined Feed
  } catch (e) {
    console.error(e)
  }
}

function anyApiJob(wallet) {
  const iface = new ethers.utils.Interface(ICNOracle__factory.abi)

  async function wrapper(job) {
    const inData: IAnyApiWorkerReporter = job.data
    console.debug('anyApiJob:inData', inData)

    try {
      const _data = typeof inData.data === 'number' ? Math.floor(inData.data) : inData.data

      const payload = iface.encodeFunctionData('fulfillOracleRequest', [
        inData.requestId,
        inData.callbackAddress,
        inData.callbackFunctionId,
        _data
      ])

      await sendTransaction(wallet, inData.oracleCallbackAddress, payload)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function vrfJob(wallet) {
  const iface = new ethers.utils.Interface(VRFCoordinator__factory.abi)
  const gasLimit = 3_000_000 // FIXME

  async function wrapper(job) {
    const inData: IVrfWorkerReporter = job.data
    console.debug('vrfJob:inData', inData)

    try {
      const rc: RequestCommitment = [
        inData.blockNum,
        inData.subId,
        inData.callbackGasLimit,
        inData.numWords,
        inData.sender
      ]
      console.debug('vrfJob:rc', rc)

      const proof: Proof = [
        inData.pk,
        inData.proof,
        inData.preSeed,
        inData.uPoint,
        inData.vComponents
      ]
      console.debug('vrfJob:proof', proof)

      const payload = iface.encodeFunctionData('fulfillRandomWords', [proof, rc])
      await sendTransaction(wallet, inData.callbackAddress, payload, gasLimit)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
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

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
