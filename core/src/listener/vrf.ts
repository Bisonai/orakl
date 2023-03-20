import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { VRFCoordinator__factory } from '@bisonai/orakl-contracts'
import { listen } from './listener'
import { State } from './state'
import { IListenerConfig, IRandomWordsRequested, IVrfListenerWorker } from '../types'
import {
  WORKER_VRF_QUEUE_NAME,
  CHAIN,
  VRF_LISTENER_STATE_NAME as listenerStateName
} from '../settings'
import { getVrfConfig } from '../api'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url
const { keyHash: KEY_HASH } = await getVrfConfig({ chain: CHAIN })

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const queueName = WORKER_VRF_QUEUE_NAME
  const service = 'VRF'
  const chain = CHAIN

  const state = new State({ redisClient, listenerStateName, service, chain, logger })
  // Previously stored listener config is ignored,
  // and replaced with the latest state of Orakl Network.
  await state.init()

  const listenFn = listen({
    queueName,
    processEventFn: processEvent,
    abi: VRFCoordinator__factory.abi,
    redisClient,
    logger
  })

  for (const listener of config) {
    await state.add(listener.id)
    const intervalId = await listenFn(listener)
    await state.update(listener.id, intervalId)
  }

  watchman({ listenFn, state, logger })
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as IRandomWordsRequested
    logger.debug(eventData, 'eventData')

    if (eventData.keyHash != KEY_HASH) {
      logger.info(`Ignore event with keyhash [${eventData.keyHash}]`)
    } else {
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
          age: 1_800
        }
      })
    }
  }

  return wrapper
}
