import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts'
// import { Event } from './event'
import { IListenerConfig, IDataRequested, IRequestResponseListenerWorker } from '../types'
import { WORKER_REQUEST_RESPONSE_QUEUE_NAME } from '../settings'

const FILE_NAME = import.meta.url

export function buildListener(config: IListenerConfig[], logger: Logger) {
  const queueName = WORKER_REQUEST_RESPONSE_QUEUE_NAME
  // FIXME remove loop and listen on multiple contract for the same event
  // for (const c of config) {
  // new Event(queueName, processEvent, RequestResponseCoordinator__factory.abi, c, logger).listen()
  // }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: FILE_NAME })

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
