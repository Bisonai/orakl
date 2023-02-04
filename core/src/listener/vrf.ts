import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { VRFCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { Event } from './event'
import { IListenerConfig, IRandomWordsRequested, IVrfListenerWorker } from '../types'

export function buildVrfListener(queueName: string, config: IListenerConfig[], logger: Logger) {
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processVrfEvent, VRFCoordinator__factory.abi, c, logger).listen()
  }
}

function processVrfEvent(iface: ethers.utils.Interface, queue: Queue, logger: Logger) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    logger.debug({ name: 'processVrfEvent', ...eventData }, 'eventData')

    const data: IVrfListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      blockHash: log.blockHash,
      requestId: eventData.requestId.toString(),
      seed: eventData.preSeed.toString(),
      accId: eventData.accId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      numWords: eventData.numWords,
      sender: eventData.sender,
      isDirectPayment: eventData.isDirectPayment
    }
    logger.debug({ name: 'processVrfEvent', ...data }, 'data')

    await queue.add('vrf', data, {
      jobId: data.requestId,
      removeOnComplete: {
        age: 1800 // 30 min
      }
    })
  }

  return wrapper
}
