import { Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { listenerService } from './listener'
import { State } from './state'
import { ProcessEventOutputType } from './types'
import { IListenerConfig, INewRound, IAggregatorWorker } from '../types'
import { buildSubmissionRoundJobId } from '../utils'
import { getOperatorAddress } from '../api'
import {
  WORKER_AGGREGATOR_QUEUE_NAME,
  DEPLOYMENT_NAME,
  REMOVE_ON_COMPLETE,
  CHAIN,
  DATA_FEED_LISTENER_STATE_NAME,
  DATA_FEED_SERVICE_NAME,
  AGGREGATOR_QUEUE_SETTINGS,
  LISTENER_DATA_FEED_LATEST_QUEUE_NAME,
  LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME,
  LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME
} from '../settings'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger
) {
  const stateName = DATA_FEED_LISTENER_STATE_NAME
  const service = DATA_FEED_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'NewRound'
  const latestQueueName = LISTENER_DATA_FEED_LATEST_QUEUE_NAME
  const historyQueueName = LISTENER_REQUEST_RESPONSE_HISTORY_QUEUE_NAME
  const processEventQueueName = LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = WORKER_AGGREGATOR_QUEUE_NAME
  const abi = Aggregator__factory.abi
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

  async function wrapper(log): Promise<ProcessEventOutputType> {
    const eventData = iface.parseLog(log).args as unknown as INewRound
    _logger.debug(eventData, 'eventData')

    const oracleAddress = log.address
    const roundId = eventData.roundId.toNumber()
    const operatorAddress = await getOperatorAddress({ oracleAddress, logger: _logger })

    const jobName = 'event'
    const jobId = buildSubmissionRoundJobId({
      oracleAddress,
      roundId,
      deploymentName: DEPLOYMENT_NAME
    })
    const job = {
      jobName,
      jobId
    }

    if (eventData.startedBy == operatorAddress) {
      _logger.debug(`Ignore event emitted by ${eventData.startedBy} for round ${roundId}`)
      return { ...job, jobData: null }
    } else {
      // NewRound emitted by somebody else
      const jobData: IAggregatorWorker = {
        oracleAddress,
        roundId,
        workerSource: 'event'
      }
      _logger.debug(jobData, 'jobData')

      return { ...job, jobQueueSettings: AGGREGATOR_QUEUE_SETTINGS, jobData }
    }
  }

  return wrapper
}
