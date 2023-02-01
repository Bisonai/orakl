import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { Event } from './event'
import { IListenerConfig, IDataRequested, IRequestResponseListenerWorker } from '../types'

export function buildListener(queueName: string, config: IListenerConfig[]) {
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processEvent, RequestResponseCoordinator__factory.abi, c).listen()
  }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IDataRequested
    console.debug('requestResponse:processEvent:eventData', eventData)

    const data: IRequestResponseListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId.toString(),
      accId: eventData.accId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      sender: eventData.sender,
      isDirectPayment: eventData.isDirectPayment,
      data: eventData.data.toString()
    }
    console.debug('requestResponse:processEvent:data', data)

    await queue.add('icn', data)
  }

  return wrapper
}
