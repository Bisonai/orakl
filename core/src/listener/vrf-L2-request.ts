import { Endpoint__factory } from '@bisonai/orakl-contracts'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getVrfConfig } from '../api'
import {
  CHAIN,
  L2_CHAIN,
  L2_LISTENER_VRF_REQUEST_HISTORY_QUEUE_NAME,
  L2_LISTENER_VRF_REQUEST_LATEST_QUEUE_NAME,
  L2_LISTENER_VRF_REQUEST_PROCESS_EVENT_QUEUE_NAME,
  L2_VRF_REQUEST_LISTENER_STATE_NAME,
  L2_VRF_REQUEST_SERVICE_NAME,
  L2_WORKER_VRF_REQUEST_QUEUE_NAME
} from '../settings'
import { IL2EndpointListenerWorker, IListenerConfig, IRandomWordsRequested } from '../types'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const stateName = L2_VRF_REQUEST_LISTENER_STATE_NAME
  const service = L2_VRF_REQUEST_SERVICE_NAME
  const l2Chain = L2_CHAIN
  const eventName = 'RandomWordsRequested'
  const latestQueueName = L2_LISTENER_VRF_REQUEST_LATEST_QUEUE_NAME
  const historyQueueName = L2_LISTENER_VRF_REQUEST_HISTORY_QUEUE_NAME
  const processEventQueueName = L2_LISTENER_VRF_REQUEST_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = L2_WORKER_VRF_REQUEST_QUEUE_NAME
  const abi = Endpoint__factory.abi
  const iface = new ethers.utils.Interface(abi)

  listenerService({
    config,
    abi,
    stateName,
    service,
    chain: l2Chain,
    eventName,
    latestQueueName,
    historyQueueName,
    processEventQueueName,
    workerQueueName,
    processFn: await processEvent({ iface, logger }),
    redisClient,
    listenerInitType: 'latest',
    logger
  })
}

async function processEvent({ iface, logger }: { iface: ethers.utils.Interface; logger: Logger }) {
  const _logger = logger.child({ name: 'processEvent', file: FILE_NAME })
  const { keyHash } = await getVrfConfig({ chain: CHAIN })

  async function wrapper(log): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    _logger.debug(eventData, 'eventData')

    if (eventData.keyHash != keyHash) {
      _logger.info(`Ignore event with keyhash [${eventData.keyHash}]`)
    } else {
      const jobName = 'vrf-l2-request'
      const requestId = eventData.requestId.toString()
      const jobData: IL2EndpointListenerWorker = {
        callbackAddress: log.address,
        blockNum: log.blockNumber,
        blockHash: log.blockHash,
        requestId,
        seed: eventData.preSeed.toString(),
        accId: eventData.accId.toString(),
        callbackGasLimit: eventData.callbackGasLimit,
        numWords: eventData.numWords,
        sender: eventData.sender,
        keyHash: eventData.keyHash
      }
      _logger.debug(jobData, 'jobData')

      return { jobName, jobId: requestId, jobData }
    }
  }

  return wrapper
}
