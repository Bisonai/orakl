import { L2Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  CHAIN,
  L1_ENDPOINT,
  L2_LISTENER_REQUEST_RESPONSE_REQUEST_HISTORY_QUEUE_NAME,
  L2_LISTENER_REQUEST_RESPONSE_REQUEST_LATEST_QUEUE_NAME,
  L2_LISTENER_REQUEST_RESPONSE_REQUEST_PROCESS_EVENT_QUEUE_NAME,
  L2_REQUEST_RESPONSE_REQUEST_LISTENER_STATE_NAME,
  L2_REQUEST_RESPONSE_REQUEST_SERVICE_NAME,
  L2_WORKER_REQUEST_RESPONSE_REQUEST_QUEUE_NAME,
} from '../settings'
import { IL2DataRequested, IL2RequestResponseListenerWorker, IListenerConfig } from '../types'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger,
) {
  const stateName = L2_REQUEST_RESPONSE_REQUEST_LISTENER_STATE_NAME
  const service = L2_REQUEST_RESPONSE_REQUEST_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'DataRequested'
  const latestQueueName = L2_LISTENER_REQUEST_RESPONSE_REQUEST_LATEST_QUEUE_NAME
  const historyQueueName = L2_LISTENER_REQUEST_RESPONSE_REQUEST_HISTORY_QUEUE_NAME
  const processEventQueueName = L2_LISTENER_REQUEST_RESPONSE_REQUEST_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = L2_WORKER_REQUEST_RESPONSE_REQUEST_QUEUE_NAME
  const abi = L2Endpoint__factory.abi
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
  const _logger = logger.child({ name: 'L2-Request-Response processEvent', file: FILE_NAME })

  async function wrapper(log: ethers.Event): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IL2DataRequested
    _logger.debug(eventData, 'eventData')

    const requestId = eventData.requestId.toString()
    const jobData: IL2RequestResponseListenerWorker = {
      callbackAddress: L1_ENDPOINT,
      blockNum: log.blockNumber.toString(),
      requestId,
      jobId: eventData.jobId.toString(),
      accId: eventData.accId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      sender: eventData.sender,
      numSubmission: eventData.numSubmission,
      req: eventData.req,
    }
    _logger.debug(jobData, 'jobData')

    return { jobName: 'l2-request-response', jobId: requestId, jobData }
  }

  return wrapper
}
