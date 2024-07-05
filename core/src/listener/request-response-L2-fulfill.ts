import { L1Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  CHAIN,
  L1_ENDPOINT,
  L2_LISTENER_REQUEST_RESPONSE_FULFILL_HISTORY_QUEUE_NAME,
  L2_LISTENER_REQUEST_RESPONSE_FULFILL_LATEST_QUEUE_NAME,
  L2_LISTENER_REQUEST_RESPONSE_FULFILL_PROCESS_EVENT_QUEUE_NAME,
  L2_REQUEST_RESPONSE_FULFILL_LISTENER_STATE_NAME,
  L2_REQUEST_RESPONSE_FULFILL_SERVICE_NAME,
  L2_WORKER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME,
} from '../settings'
import {
  IL2DataRequestFulfilled,
  IL2RequestResponseFulfillListenerWorker,
  IListenerConfig,
} from '../types'
import { listenerService } from './listener'
import { parseResponse } from './request-response-L2.utils'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger,
) {
  const stateName = L2_REQUEST_RESPONSE_FULFILL_LISTENER_STATE_NAME
  const service = L2_REQUEST_RESPONSE_FULFILL_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'DataRequestFulfilled'
  const latestQueueName = L2_LISTENER_REQUEST_RESPONSE_FULFILL_LATEST_QUEUE_NAME
  const historyQueueName = L2_LISTENER_REQUEST_RESPONSE_FULFILL_HISTORY_QUEUE_NAME
  const processEventQueueName = L2_LISTENER_REQUEST_RESPONSE_FULFILL_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = L2_WORKER_REQUEST_RESPONSE_FULFILL_QUEUE_NAME
  const abi = L1Endpoint__factory.abi
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
  const _logger = logger.child({
    name: 'L2-Request-Response-Fulfill processEvent',
    file: FILE_NAME,
  })

  async function wrapper(log: ethers.Event): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IL2DataRequestFulfilled
    _logger.debug(eventData, 'eventData')
    const requestId = eventData.requestId.toString()
    const response = parseResponse[eventData.jobId](eventData)
    const jobData: IL2RequestResponseFulfillListenerWorker = {
      callbackAddress: L1_ENDPOINT,
      blockNum: log.blockNumber,
      requestId,
      l2RequestId: eventData.l2RequestId.toString(),
      jobId: eventData.jobId.toString(),
      callbackGasLimit: eventData.callbackGasLimit,
      sender: eventData.sender,
      response,
    }
    _logger.debug(jobData, 'jobData')

    return { jobName: 'l2-request-response-fulfill', jobId: requestId, jobData }
  }

  return wrapper
}
