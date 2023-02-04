import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { RequestResponseCoordinator__factory } from '@bisonai-cic/icn-contracts'
import { Event } from './event'
import { IListenerConfig, IDataRequested, IRequestResponseListenerWorker } from '../types'

export function buildListener(queueName: string, config: IListenerConfig[], logger: Logger) {
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processEvent, RequestResponseCoordinator__factory.abi, c, logger).listen()
  }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: import.meta.url })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IDataRequested
    logger.debug(eventData, 'eventData')

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
    logger.debug(data, 'data')

    await queue.add('request-response', data)
  }

  return wrapper
}
