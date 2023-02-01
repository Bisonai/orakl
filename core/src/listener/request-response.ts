import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { Event } from './event'
import { IListenerConfig, INewRequest, IRequestResponseListenerWorker } from '../types'

export function buildListener(queueName: string, config: IListenerConfig[]) {
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processEvent, RequestResponseCoordinator__factory.abi, c).listen()
  }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRequest
    console.debug('requestResponse:processEvent:eventData', eventData)

    const data: IRequestResponseListenerWorker = {
      oracleCallbackAddress: log.address,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId.toString(),
      nonce: eventData.nonce.toString(),
      callbackAddress: eventData.callbackAddress.toString(),
      callbackFunctionId: eventData.callbackFunctionId.toString(),
      _data: eventData._data.toString()
    }
    console.debug('requestResponse:processEvent:data', data)

    await queue.add('icn', data)
  }

  return wrapper
}
