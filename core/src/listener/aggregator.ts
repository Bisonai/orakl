import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { Event } from './event'
import { IListenerConfig, INewRound, IAggregatorWorker } from '../types'
import { buildReporterJobId } from '../utils'
import {
  PUBLIC_KEY,
  WORKER_AGGREGATOR_QUEUE_NAME,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  REMOVE_ON_FAIL
} from '../settings'

const FILE_NAME = import.meta.url

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
      const aggregatorAddress = log.address.toLowerCase()
      const roundId = eventData.roundId.toNumber()
      // NewRound emitted by somebody else
      const data: IAggregatorWorker = {
        aggregatorAddress,
        roundId,
        workerSource: 'event'
      }
      logger.debug(data, 'data')

      await queue.add('aggregator', data, {
        removeOnComplete: REMOVE_ON_COMPLETE,
        removeOnFail: REMOVE_ON_FAIL,
        jobId: buildReporterJobId({ aggregatorAddress, roundId, deploymentName: DEPLOYMENT_NAME })
      })
    }
  }

  return wrapper
}
