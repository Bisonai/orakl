import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import {
  INewRequest,
  IRandomWordsRequested,
  IAnyApiListenerWorker,
  IVrfListenerWorker
} from '../types'

export function processICNEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRequest
    console.debug('processICNEvent:eventData', eventData)
    const data: IAnyApiListenerWorker = {
      oracleCallbackAddress: log.address,
      requestId: eventData.requestId.toString(),
      jobId: eventData.jobId.toString(),
      nonce: eventData.nonce.toString(),
      callbackAddress: eventData.callbackAddress.toString(),
      callbackFunctionId: eventData.callbackFunctionId.toString(),
      _data: eventData._data.toString()
    }
    console.debug('processICNEvent:data', data)

    await queue.add('icn', data)
  }
  return wrapper
}

export function processVrfEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    console.debug('processVrfEvent:eventData', eventData)
    const data: IVrfListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      blockHash: log.blockHash,
      requestId: eventData.requestId.toString(),
      seed: eventData.preSeed.toString(),
      subId: eventData.subId.toString(),
      minimumRequestConfirmations: eventData.minimumRequestConfirmations,
      callbackGasLimit: eventData.callbackGasLimit,
      numWords: eventData.numWords,
      sender: eventData.sender
    }
    console.debug('processVrfEvent:data', data)

    await queue.add('vrf', data)
  }
  return wrapper
}
