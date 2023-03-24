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
  REMOVE_ON_FAIL,
  CHAIN,
  DATA_FEED_LISTENER_STATE_NAME,
  DATA_FEED_SERVICE_NAME
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

      await queue.add('event', data, {
        jobId: buildSubmissionRoundJobId({
          oracleAddress,
          roundId,
          deploymentName: DEPLOYMENT_NAME
        }),
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
      logger.debug({ job: 'event-added' }, 'job-added')
    }
  }

  return wrapper
}
