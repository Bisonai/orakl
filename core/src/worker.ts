import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { ethers } from 'ethers'
import { Worker, Queue } from 'bullmq'
import { got } from 'got'
import { IcnError, IcnErrorCode } from './errors'
import {
  IAdapter,
  IVrfResponse,
  IPredefinedFeedListenerWorker,
  IAnyApiListenerWorker,
  IVrfListenerWorker,
  IAnyApiWorkerReporter,
  IPredefinedFeedWorkerReporter,
  IVrfWorkerReporter
} from './types'
import { loadJson, pipe, remove0x, readFromJson } from './utils'
import { reducerMapping } from './reducer'
import {
  localAggregatorFn,
  WORKER_ANY_API_QUEUE_NAME,
  REPORTER_ANY_API_QUEUE_NAME,
  WORKER_PREDEFINED_FEED_QUEUE_NAME,
  REPORTER_PREDEFINED_FEED_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME,
  BULLMQ_CONNECTION,
  ADAPTER_ROOT_DIR
} from './settings'
import { decodeAnyApiRequest } from './decoding'
import { prove, decode, getFastVerifyComponents } from './vrf/index'
import { VRF_SK, VRF_PK, VRF_PK_X, VRF_PK_Y } from './load-parameters'
import express from 'express'
import { healthCheck } from './healthchecker'

async function main() {
  const adapters = (await loadAdapters())[0] // FIXME take all adapters
  console.debug('main:adapters', adapters)

  // TODO if job not finished, return job in queue

  new Worker(WORKER_ANY_API_QUEUE_NAME, anyApiJob(REPORTER_ANY_API_QUEUE_NAME), BULLMQ_CONNECTION)

  new Worker(
    WORKER_PREDEFINED_FEED_QUEUE_NAME,
    predefinedFeedJob(REPORTER_PREDEFINED_FEED_QUEUE_NAME, adapters),
    BULLMQ_CONNECTION
  )

  new Worker(WORKER_VRF_QUEUE_NAME, vrfJob(REPORTER_VRF_QUEUE_NAME), BULLMQ_CONNECTION)

  // simple health check, later readness, liveness?
  const server = express()
  server.get('/health-check', (_, res) => {
    res.send(healthCheck())
  })
  server.listen(8030)
}

function extractFeeds(adapter) {
  const adapterId = adapter.adapter_id
  const feeds = adapter.feeds.map((f) => {
    const reducers = f.reducers?.map((r) => {
      // TODO check if exists
      return reducerMapping[r.function](r.args)
    })

    return {
      url: f.url,
      headers: f.headers,
      method: f.method,
      reducers
    }
  })

  return { [adapterId]: feeds }
}

async function loadAdapters() {
  const adapterPaths = await Fs.readdir(ADAPTER_ROOT_DIR)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(ADAPTER_ROOT_DIR, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)
  return activeRawAdapters.map((a) => extractFeeds(a))
}

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['active', 'name', 'job_type', 'adapter_id', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) =>
    Object.prototype.hasOwnProperty.call(adapter, p)
  )
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new IcnError(IcnErrorCode.InvalidOperator)
  }
}

function anyApiJob(queueName) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IAnyApiListenerWorker = job.data
    console.debug('anyApiJob:inData', inData)

    try {
      const res = await processAnyApiRequest(inData._data)

      const outData: IAnyApiWorkerReporter = {
        oracleCallbackAddress: inData.oracleCallbackAddress,
        requestId: inData.requestId,
        jobId: inData.jobId,
        callbackAddress: inData.callbackAddress,
        callbackFunctionId: inData.callbackFunctionId,
        data: res
      }
      console.debug('anyApiJob:outData', outData)

      await queue.add('any-api', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

async function processAnyApiRequest(reqEnc: string): Promise<string | number> {
  const req = decodeAnyApiRequest(reqEnc)
  console.debug('processAnyApiRequest:req', req)

  let res: string = await got(req.get).json()
  if (req.path) {
    res = readFromJson(res, req.path)
  }

  console.debug('processAnyApiRequest:res', res)
  return res
}

function predefinedFeedJob(queueName, adapters) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IPredefinedFeedListenerWorker = job.data
    console.debug('predefinedFeedJob:inData', inData)

    // FIXME take adapterId from job.data (information emitted by on-chain event)
    try {
      const allResults = await Promise.all(
        adapters[inData.jobId].map(async (adapter) => {
          const options = {
            method: adapter.method,
            headers: adapter.headers
          }

          try {
            const rawData = await got(adapter.url, options).json()
            return pipe(...adapter.reducers)(rawData)
          } catch (e) {
            console.error(e)
          }
        })
      )
      console.debug('predefinedFeedJob:allResults', allResults)

      // FIXME single node aggregation of multiple results
      // FIXME check if aggregator is defined and if exists
      const res = localAggregatorFn(...allResults)
      console.debug('predefinedFeedJob:res', res)

      const outData: IPredefinedFeedWorkerReporter = {
        requestId: inData.requestId,
        jobId: inData.jobId,
        callbackAddress: inData.callbackAddress,
        callbackFunctionId: inData.callbackFunctionId,
        data: res
      }
      console.debug('predefinedFeedJob:outData', outData)

      await queue.add('predefined-feed', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function vrfJob(queueName) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IVrfListenerWorker = job.data
    console.debug('vrfJob:inData', inData)

    try {
      const alpha = remove0x(
        ethers.utils.solidityKeccak256(['uint256', 'bytes32'], [inData.seed, inData.blockHash])
      )

      console.debug('vrfJob:alpha', alpha)
      const { pk, proof, uPoint, vComponents } = processVrfRequest(alpha)

      const outData: IVrfWorkerReporter = {
        callbackAddress: inData.callbackAddress,
        blockNum: inData.blockNum,
        requestId: inData.requestId,
        seed: inData.seed,
        subId: inData.subId,
        minimumRequestConfirmations: inData.minimumRequestConfirmations,
        callbackGasLimit: inData.callbackGasLimit,
        numWords: inData.numWords,
        sender: inData.sender,
        pk,
        proof,
        preSeed: inData.seed,
        uPoint,
        vComponents
      }
      console.debug('vrfJob:outData', outData)

      await queue.add('vrf', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

function processVrfRequest(alpha: string): IVrfResponse {
  console.debug('processVrfRequest:alpha', alpha)

  const proof = prove(VRF_SK, alpha)
  const [Gamma, c, s] = decode(proof)
  const fast = getFastVerifyComponents(VRF_PK, proof, alpha)

  if (fast == 'INVALID') {
    console.error('INVALID')
    // TODO throw more specific error
    throw Error()
  }

  return {
    pk: [VRF_PK_X, VRF_PK_Y],
    proof: [Gamma.x.toString(), Gamma.y.toString(), c.toString(), s.toString()],
    uPoint: [fast.uX, fast.uY],
    vComponents: [fast.sHX, fast.sHY, fast.cGX, fast.cGY]
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
