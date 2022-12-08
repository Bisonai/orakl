import * as Fs from 'node:fs/promises'
import * as Path from 'node:path'
import { ethers } from 'ethers'
import { BN } from 'bn.js'
import { Worker, Queue } from 'bullmq'
import { got, Options } from 'got'
import { IcnError, IcnErrorCode } from './errors'
import { IAdapter, IVrfRequest, IVrfResponse } from './types'
import { buildBullMqConnection, loadJson, pipe } from './utils'
import { buildAdapterRootDir, readFromJson } from './utils'
import { reducerMapping } from './reducer'
import {
  localAggregatorFn,
  WORKER_REQUEST_QUEUE_NAME,
  REPORTER_REQUEST_QUEUE_NAME,
  WORKER_VRF_QUEUE_NAME,
  REPORTER_VRF_QUEUE_NAME
} from './settings'
import { decodeAnyApiRequest } from './decoding'
import { prove, verify } from './vrf/index'
import { VRF_SK, VRF_PK } from './load-parameters'

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
  const adapterRootDir = buildAdapterRootDir()
  const adapterPaths = await Fs.readdir(adapterRootDir)

  const allRawAdapters = await Promise.all(
    adapterPaths.map(async (ap) => validateAdapter(await loadJson(Path.join(adapterRootDir, ap))))
  )
  const activeRawAdapters = allRawAdapters.filter((a) => a.active)
  return activeRawAdapters.map((a) => extractFeeds(a))
}

function validateAdapter(adapter): IAdapter {
  // TODO extract properties from Interface
  const requiredProperties = ['active', 'name', 'job_type', 'adapter_id', 'feeds']
  // TODO show where is the error
  const hasProperty = requiredProperties.map((p) => adapter.hasOwnProperty(p))
  const isValid = hasProperty.every((x) => x)

  if (isValid) {
    return adapter as IAdapter
  } else {
    throw new IcnError(IcnErrorCode.InvalidOperator)
  }
}

async function processAnyApi(apiRequest) {
  console.log('Any API', apiRequest)

  const request = decodeAnyApiRequest(apiRequest)
  let data: any = await got(request.get).json()

  if (request.path) {
    data = readFromJson(data, request.path)
  }

  return data
}

function processVrfRequest(vrfRequest: IVrfRequest): IVrfResponse {
  console.log('VRF Request')

  const proof = prove(VRF_SK, vrfRequest.alpha)
  // TODO remove status
  const [status, beta] = verify(VRF_PK, proof, vrfRequest.alpha)

  // TODO make sure beta is string
  if (beta == null) {
    throw Error()
  }

  return {
    proof,
    beta
  }
}

function vrfJob(queue) {
  async function wrapper(job) {
    const data = job.data
    console.log('VRF request', data)

    try {
      const seed = new BN(data.seed.hex.slice(2), 'hex') // FIXME
      const alpha = ethers.utils.solidityKeccak256(
        ['uint256', 'bytes32'],
        [seed.toString(), data.blockHash]
      )

      console.log('alpha', alpha)
      const { proof, beta } = processVrfRequest({ alpha })
      console.log(`proof ${proof}`)
      console.log(`beta ${beta}`)

      await queue.add('report', {
        blockNum: data.blockNum,
        requestId: data.requestId,
        seed: data.seed,
        subId: data.subId,
        minimumRequestConfirmations: data.minimumRequestConfirmations,
        callbackGasLimit: data.callbackGasLimit,
        numWords: data.numWords,
        sender: data.sender,
        proof,
        beta
      })
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

async function main() {
  const adapters = (await loadAdapters())[0] // FIXME take all adapters
  console.log('adapters', adapters)

  const reporterRequestQueue = new Queue(REPORTER_REQUEST_QUEUE_NAME, buildBullMqConnection())
  const reporterVrfQueue = new Queue(REPORTER_VRF_QUEUE_NAME, buildBullMqConnection())

  // TODO if job not finished, return job in queue

  const vrfWorker = new Worker(
    WORKER_VRF_QUEUE_NAME,
    vrfJob(reporterVrfQueue),
    buildBullMqConnection()
  )

  const requestWorker = new Worker(
    WORKER_REQUEST_QUEUE_NAME,
    async (job) => {
      console.log('Any API request', job.data)

      const {
        requestId,
        jobId,
        nonce,
        callbackAddress,
        callbackFunctionId,
        _data: dataRequest
      } = job.data

      let data // FIXME

      if (jobId == '0x0000000000000000000000000000000000000000000000000000000000000000') {
        console.log('Predefined feed')

        // FIXME take adapterId from job.data (information emitted by on-chain event)
        const results = await Promise.all(
          adapters[jobId].map(async (adapter) => {
            console.log('current adapter', adapter)

            const options = {
              method: adapter.method,
              headers: adapter.headers
            }

            try {
              const rawData = await got(adapter.url, options).json()
              return pipe(...adapter.reducers)(rawData)
              // console.log(`data ${data}`)
            } catch (e) {
              // FIXME
              console.error(e)
            }
          })
        )
        console.log(`results ${results}`)

        // FIXME single node aggregation of multiple results
        // FIXME check if aggregator is defined and if exists
        try {
          data = localAggregatorFn(...results)
          console.log(`data ${data}`)
        } catch (e) {
          console.error(e)
        }
      } else if (jobId == '0x1111111111111111111111111111111111111111111111111111111111111111') {
        // VRF
        const res = processVrfRequest({ alpha: requestId })
      } else {
        data = processAnyApi(dataRequest)
      }

      await reporterRequestQueue.add('report', {
        requestId,
        jobId,
        callbackAddress,
        callbackFunctionId,
        data
      })
    },
    buildBullMqConnection()
  )
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
