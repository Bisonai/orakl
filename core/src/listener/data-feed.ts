import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { listen } from './listener'
import { State } from './state'
import { IListenerConfig, INewRound, IAggregatorWorker } from '../types'
import { buildSubmissionRoundJobId } from '../utils'
import {
  PUBLIC_KEY,
  WORKER_AGGREGATOR_QUEUE_NAME,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  CHAIN,
  DATA_FEED_LISTENER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  AGGREGATOR_QUEUE_SETTINGS
} from '../settings'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const queueName = WORKER_AGGREGATOR_QUEUE_NAME

  const state = new State({
    redisClient,
    stateName: DATA_FEED_LISTENER_STATE_NAME,
    service: DATA_FEED_SERVICE_NAME,
    chain: CHAIN,
    logger
  })
  await state.clear()

  const listenFn = listen({
    queueName,
    processEventFn: processEvent,
    abi: Aggregator__factory.abi,
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
  const aggregatorQueue = queue

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRound
    logger.debug(eventData, 'eventData')

    if (eventData.startedBy != PUBLIC_KEY) {
      const oracleAddress = log.address.toLowerCase()
      const roundId = eventData.roundId.toNumber()
      // NewRound emitted by somebody else
      const data: IAggregatorWorker = {
        oracleAddress,
        roundId,
        workerSource: 'event'
      }
      logger.debug(data, 'data')

      const jobId = buildSubmissionRoundJobId({
        oracleAddress,
        roundId,
        deploymentName: DEPLOYMENT_NAME
      })
      await aggregatorQueue.add('event', data, {
        jobId,
        removeOnComplete: REMOVE_ON_COMPLETE,
        ...AGGREGATOR_QUEUE_SETTINGS
      })
      logger.debug({ job: 'event-added' }, `Listener submitted job with ID=${jobId}`)
    }
  }

  return wrapper
}
