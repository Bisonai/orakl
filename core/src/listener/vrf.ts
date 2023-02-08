import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { Event } from './event'
import { IListenerConfig, IRandomWordsRequested, IVrfListenerWorker } from '../types'
import { WORKER_VRF_QUEUE_NAME } from '../settings'

const FILE_NAME = import.meta.url

export function buildListener(config: IListenerConfig[], logger: Logger) {
  const queueName = WORKER_VRF_QUEUE_NAME
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processEvent, VRFCoordinator__factory.abi, c, logger).listen()
  }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    logger.debug(eventData, 'eventData')

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
    logger.debug(data, 'data')

    await queue.add('vrf', data, {
      jobId: data.requestId,
      removeOnComplete: {
        age: 1800 // 30 min
      }
    })
  }

  return wrapper
}
