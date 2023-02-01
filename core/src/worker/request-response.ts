import { Worker, Queue } from 'bullmq'
import axios from 'axios'
import { IRequestResponseListenerWorker, IRequestResponseWorkerReporter } from '../types'
import { readFromJson } from '../utils'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  REPORTER_REQUEST_RESPONSE_QUEUE_NAME,
  BULLMQ_CONNECTION
} from '../settings'
import { decodeRequest } from '../decoding'

export async function worker() {
  console.debug('requestResponse:worker')
  new Worker(
    WORKER_REQUEST_RESPONSE_QUEUE_NAME,
    job(REPORTER_REQUEST_RESPONSE_QUEUE_NAME),
    BULLMQ_CONNECTION
  )
}

function job(queueName) {
  const queue = new Queue(queueName, BULLMQ_CONNECTION)

  async function wrapper(job) {
    const inData: IRequestResponseListenerWorker = job.data
    console.debug('requestResponseJob:inData', inData)

    try {
      const res = await processRequest(inData._data)

      const outData: IRequestResponseWorkerReporter = {
        oracleCallbackAddress: inData.oracleCallbackAddress,
        requestId: inData.requestId,
        jobId: inData.jobId,
        callbackAddress: inData.callbackAddress,
        callbackFunctionId: inData.callbackFunctionId,
        data: res
      }
      console.debug('requestResponseJob:outData', outData)

      await queue.add('request-response', outData)
    } catch (e) {
      console.error(e)
    }
  }

  return wrapper
}

async function processRequest(reqEnc: string): Promise<string | number> {
  const req = decodeRequest(reqEnc)
  console.debug('processRequest:req', req)

  let res: string = (await axios.get(req.get)).data
  if (req.path) {
    res = readFromJson(res, req.path)
  }

  console.debug('processRequest:res', res)
  return res
}
