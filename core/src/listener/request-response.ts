import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts'
import { listenerService } from './listener'
import { IListenerConfig, IDataRequested, IRequestResponseListenerWorker } from '../types'
import {
  BULLMQ_CONNECTION,
  CHAIN,
  LISTENER_JOB_SETTINGS,
  LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME,
  REQUEST_RESPONSE_LISTENER_STATE_NAME,
  REQUEST_RESPONSE_SERVICE_NAME,
  WORKER_REQUEST_RESPONSE_QUEUE_NAME
} from '../settings'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const stateName = REQUEST_RESPONSE_LISTENER_STATE_NAME
  const service = REQUEST_RESPONSE_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'DataRequested'
  const latestQueueName = LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME
  const historyQueueName = LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME
  const processEventQueueName = LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME
  const abi = RequestResponseCoordinator__factory.abi
  const iface = new ethers.utils.Interface(abi)

  listenerService({
    config,
    abi,
    stateName,
    service,
    chain,
    eventName,
    latestQueueName,
    historyQueueName,
    processEventQueueName,
    processFn: await processEvent({ iface, logger }),
    redisClient,
    logger
  })
}

async function processEvent({ iface, logger }: { iface: ethers.utils.Interface; logger: Logger }) {
  const _logger = logger.child({ name: 'Request-Response processEvent', file: FILE_NAME })
  const workerQueue = new Queue(WORKER_REQUEST_RESPONSE_QUEUE_NAME, BULLMQ_CONNECTION)

  async function wrapper(log: ethers.Event) {
    const eventData = iface.parseLog(log).args as unknown as IDataRequested
    _logger.debug(eventData, 'eventData')

    const requestId = eventData.requestId.toString()
    const outData: IRequestResponseListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      requestId,
      jobId: eventData.jobId.toString(),
      accId: eventData.accId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      sender: eventData.sender,
      isDirectPayment: eventData.isDirectPayment,
      data: eventData.data.toString()
    }
    _logger.debug(outData, 'outData')

    await workerQueue.add('request-response', outData, {
      jobId: requestId,
      ...LISTENER_JOB_SETTINGS
    })
  }

  return wrapper
}
