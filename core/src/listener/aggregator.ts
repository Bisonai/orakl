import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'
import { Event } from './event'
import { IListenerConfig, INewRound, IAggregatorListenerWorker } from '../types'
import { PUBLIC_KEY } from '../settings'

export function buildAggregatorListener(
  queueName: string,
  config: IListenerConfig[],
  logger: Logger
) {
  // FIXME remove loop and listen on multiple contract for the same event
  for (const c of config) {
    new Event(queueName, processAggregatorEvent, Aggregator__factory.abi, c, logger).listen()
  }
}

function processAggregatorEvent(iface: ethers.utils.Interface, queue: Queue, _logger: Logger) {
  const logger = _logger.child({ name: 'processAggregatorEvent', file: import.meta.url })

  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRound
    logger.debug(eventData, 'eventData')

    if (eventData.startedBy != PUBLIC_KEY) {
      // NewRound emitted by somebody else
      const data: IAggregatorListenerWorker = {
        address: log.address,
        roundId: eventData.roundId,
        startedBy: eventData.startedBy,
        startedAt: eventData.startedAt
      }
      logger.debug(data, 'data')

      await queue.add('aggregator', data, { removeOnComplete: true })
    }
  }

  return wrapper
}
