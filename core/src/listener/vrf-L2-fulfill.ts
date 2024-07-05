import { L1Endpoint__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  CHAIN,
  L2_ENDPOINT,
  L2_LISTENER_VRF_FULFILL_HISTORY_QUEUE_NAME,
  L2_LISTENER_VRF_FULFILL_LATEST_QUEUE_NAME,
  L2_LISTENER_VRF_FULFILL_PROCESS_EVENT_QUEUE_NAME,
  L2_VRF_FULFILL_LISTENER_STATE_NAME,
  L2_VRF_FULFILL_SERVICE_NAME,
  L2_WORKER_VRF_FULFILL_QUEUE_NAME,
} from '../settings'
import { IL2VrfFulfillListenerWorker, IListenerConfig, IRandomWordsFulfilled } from '../types'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger,
) {
  const stateName = L2_VRF_FULFILL_LISTENER_STATE_NAME
  const service = L2_VRF_FULFILL_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'RandomWordFulfilled'
  const latestQueueName = L2_LISTENER_VRF_FULFILL_LATEST_QUEUE_NAME
  const historyQueueName = L2_LISTENER_VRF_FULFILL_HISTORY_QUEUE_NAME
  const processEventQueueName = L2_LISTENER_VRF_FULFILL_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = L2_WORKER_VRF_FULFILL_QUEUE_NAME
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
  const _logger = logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsFulfilled
    _logger.debug(eventData, 'eventData')
    const jobName = 'vrf-l2-fulfill'
    const requestId = eventData.requestId.toString()
    const jobData: IL2VrfFulfillListenerWorker = {
      callbackAddress: L2_ENDPOINT,
      blockNum: log.blockNumber,
      blockHash: log.blockHash,
      requestId: eventData.requestId.toString(),
      sender: eventData.sender,
      l2RequestId: eventData.l2RequestId.toString(),
      randomWords: eventData.randomWords.map((m) => m.toString()),
      callbackGasLimit: eventData.callbackGasLimit,
    }

    _logger.debug(jobData, 'jobData')

    return { jobName, jobId: requestId, jobData }
  }

  return wrapper
}
