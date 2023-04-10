import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { State } from './state'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'
import { IListenerConfig, IRandomWordsRequested, IVrfListenerWorker } from '../types'
import {
  WORKER_VRF_QUEUE_NAME,
  CHAIN,
  VRF_SERVICE_NAME,
  VRF_LISTENER_STATE_NAME,
  LISTENER_VRF_LATEST_QUEUE_NAME,
  LISTENER_VRF_HISTORY_QUEUE_NAME,
  LISTENER_VRF_PROCESS_EVENT_QUEUE_NAME
} from '../settings'
import { getVrfConfig } from '../api'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const stateName = VRF_LISTENER_STATE_NAME
  const service = VRF_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'RandomWordsRequested'
  const latestQueueName = LISTENER_VRF_LATEST_QUEUE_NAME
  const historyQueueName = LISTENER_VRF_HISTORY_QUEUE_NAME
  const processEventQueueName = LISTENER_VRF_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = WORKER_VRF_QUEUE_NAME
  const abi = VRFCoordinator__factory.abi
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
    logger
  })
}

async function processEvent({ iface, logger }: { iface: ethers.utils.Interface; logger: Logger }) {
  const _logger = logger.child({ name: 'processEvent', file: FILE_NAME })
  const { keyHash } = await getVrfConfig({ chain: CHAIN })

  async function wrapper(log): Promise<ProcessEventOutputType> {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    _logger.debug(eventData, 'eventData')

    const requestId = eventData.requestId.toString()
    const jobName = 'vrf'
    const job = {
      jobName,
      jobId: requestId
    }

    if (eventData.keyHash != keyHash) {
      _logger.info(`Ignore event with keyhash [${eventData.keyHash}]`)
      return { ...job, jobData: null }
    } else {
      const jobData: IVrfListenerWorker = {
        callbackAddress: log.address,
        blockNum: log.blockNumber,
        blockHash: log.blockHash,
        requestId,
        seed: eventData.preSeed.toString(),
        accId: eventData.accId.toString(),
        callbackGasLimit: eventData.callbackGasLimit,
        numWords: eventData.numWords,
        sender: eventData.sender,
        isDirectPayment: eventData.isDirectPayment
      }
      _logger.debug(jobData, 'jobData')

      return { ...job, jobData }
    }
  }

  return wrapper
}
