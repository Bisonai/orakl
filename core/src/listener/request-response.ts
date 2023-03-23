import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts'
import { listen } from './listener'
import { State } from './state'
import { IListenerConfig, IDataRequested, IRequestResponseListenerWorker } from '../types'
import {
  WORKER_REQUEST_RESPONSE_QUEUE_NAME,
  CHAIN,
  REQUEST_RESPONSE_LISTENER_STATE_NAME,
  REQUEST_RESPONSE_SERVICE_NAME
} from '../settings'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const queueName = WORKER_REQUEST_RESPONSE_QUEUE_NAME

  const state = new State({
    redisClient,
    stateName: REQUEST_RESPONSE_LISTENER_STATE_NAME,
    service: REQUEST_RESPONSE_SERVICE_NAME,
    chain: CHAIN,
    logger
  })
  await state.clear()

  const listenFn = listen({
    queueName,
    processEventFn: processEvent,
    abi: RequestResponseCoordinator__factory.abi,
    redisClient,
    logger
  })

  for (const listener of config) {
    await state.add(listener.id)
    const intervalId = await listenFn(listener)
    await state.update(listener.id, intervalId)
  }

  await watchman({ listenFn, state, logger })
}

async function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
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

    await queue.add('request-response', data, {
      jobId: data.requestId,
      removeOnComplete: {
        age: 1_800
      }
    })
  }

  return wrapper
}
