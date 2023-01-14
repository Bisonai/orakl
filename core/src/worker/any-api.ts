import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { IAnyApiListenerWorker, IAnyApiWorkerReporter } from '../types'
import { readFromJson } from '../utils'
import {
  WORKER_ANY_API_QUEUE_NAME,
  REPORTER_ANY_API_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'
import { decodeAnyApiRequest } from '../decoding'

export async function anyApiWorker() {
  console.debug('anyApiWorker')
  new Worker(WORKER_ANY_API_QUEUE_NAME, anyApiJob(REPORTER_ANY_API_QUEUE_NAME), BULLMQ_CONNECTION)
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

  let res: string = (await axios.get(req.get)).data
  if (req.path) {
    res = readFromJson(res, req.path)
  }

  console.debug('processAnyApiRequest:res', res)
  return res
}
