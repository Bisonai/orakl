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
    console.debug('requestResponse:job:inData', inData)

    try {
      const res = await processRequest(inData.data)

      const outData: IRequestResponseWorkerReporter = {
        callbackAddress: inData.callbackAddress,
        blockNum: inData.blockNum,
        requestId: inData.requestId,
        jobId: inData.jobId,
        accId: inData.accId,
        callbackGasLimit: inData.callbackGasLimit,
        sender: inData.sender,
        isDirectPayment: inData.isDirectPayment,
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
  console.debug('requestResponse:processRequest:req', req)

  let res: string = (await axios.get(req.get)).data
  if (req.path) {
    res = readFromJson(res, req.path)
  }

  console.debug('requestResponse:processRequest:res', res)
  return res
}
