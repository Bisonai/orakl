import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  CHAIN,
  LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME,
  REQUEST_RESPONSE_LISTENER_STATE_NAME,
  REQUEST_RESPONSE_SERVICE_NAME,
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
} from '../settings'
import { IDataRequested, IListenerConfig, IRequestResponseListenerWorker } from '../types'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger,
) {
  const stateName = REQUEST_RESPONSE_LISTENER_STATE_NAME
  const service = REQUEST_RESPONSE_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'DataRequested'
  const latestQueueName = LISTENER_REQUEST_RESPONSE_LATEST_QUEUE_NAME
  const historyQueueName = LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME
  const processEventQueueName = LISTENER_REQUEST_RESPONSE_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = WORKER_REQUEST_RESPONSE_QUEUE_NAME
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
    workerQueueName,
    processFn: await processEvent({ iface, logger }),
    redisClient,
    listenerInitType: 'latest',
    logger,
  })
}

async function processEvent({ iface, logger }: { iface: ethers.utils.Interface; logger: Logger }) {
  const _logger = logger.child({ name: 'Request-Response processEvent', file: FILE_NAME })

  async function wrapper(log: ethers.Event): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IDataRequested
    _logger.debug(eventData, 'eventData')

    const requestId = eventData.requestId.toString()
    const jobData: IRequestResponseListenerWorker = {
      callbackAddress: log.address,
      blockNum: log.blockNumber,
      requestId,
      jobId: eventData.jobId.toString(),
      accId: eventData.accId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      sender: eventData.sender,
      isDirectPayment: eventData.isDirectPayment,
      numSubmission: eventData.numSubmission,
      data: eventData.data.toString(),
    }
    _logger.debug(jobData, 'jobData')

    return { jobName: 'request-response', jobId: requestId, jobData }
  }

  return wrapper
}
