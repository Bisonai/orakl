import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Event } from './event'
import { IListenerConfig, INewRequest, IAnyApiListenerWorker } from '../types'

export function buildAnyApiListener(queueName: string, config: IListenerConfig) {
  new Event(queueName, processAnyApiEvent, config).listen()
}

export function processAnyApiEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRequest
    console.debug('processAnyApiEvent:eventData', eventData)

    const data: IAnyApiListenerWorker = {
      oracleCallbackAddress: log.address,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId.toString(),
      nonce: eventData.nonce.toString(),
      callbackAddress: eventData.callbackAddress.toString(),
      callbackFunctionId: eventData.callbackFunctionId.toString(),
      _data: eventData._data.toString()
    }
    console.debug('processAnyApiEvent:data', data)

    await queue.add('icn', data)
  }

  return wrapper
}
