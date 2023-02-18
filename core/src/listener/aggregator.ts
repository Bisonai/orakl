import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { Event } from './event'
import { IListenerConfig, INewRound, IAggregatorListenerWorker } from '../types'
import { PUBLIC_KEY, WORKER_AGGREGATOR_QUEUE_NAME } from '../settings'

const FILE_NAME = import.meta.url

// FIXME move to settings?
const REMOVE_ON_COMPLETE = 500
const REMOVE_ON_FAIL = 1_000

export function buildListener(config: IListenerConfig[], logger: Logger) {
  const queueName = WORKER_AGGREGATOR_QUEUE_NAME
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processEvent, Aggregator__factory.abi, c, logger).listen()
  }
}

function processEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRound
    logger.debug(eventData, 'eventData')

    if (eventData.startedBy != PUBLIC_KEY) {
      // NewRound emitted by somebody else
      const data: IAggregatorListenerWorker = {
        address: log.address.toLowerCase(),
        roundId: eventData.roundId.toNumber(),
        startedBy: eventData.startedBy,
        startedAt: eventData.startedAt
      }
      logger.debug(data, 'data')

      await queue.add('aggregator', data, {
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL
      })
    }
  }

  return wrapper
}
