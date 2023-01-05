import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Event } from './event'
import { IListenerConfig, INewRound, IAggregatorListenerWorker } from '../types'

export function buildAggregatorListener(queueName: string, config: IListenerConfig) {
  new Event(queueName, processAggregatorEvent, config).listen()
}

function processAggregatorEvent(iface: ethers.utils.Interface, queue: Queue) {
  async function wrapper(log) {
    const eventData = iface.parseLog(log).args as unknown as INewRound
    console.debug('processAggregatorEvent:eventData', eventData)

    // TODO if I have emitted the event, then ignore

    const data: IAggregatorListenerWorker = {
      aggregatorAddress: log.address,
      roundId: eventData.roundId,
      startedBy: eventData.startedBy,
      startedAt: eventData.startedAt
    }
    console.debug('processAggregatorEvent:data', data)

    await queue.add('aggregator', data)
  }

  return wrapper
}
