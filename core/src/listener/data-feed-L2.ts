import { Aggregator__factory } from '@bisonai/orakl-contracts/v0.1'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  AGGREGATOR_QUEUE_SETTINGS,
  CHAIN,
  DEPLOYMENT_NAME,
  L2_DATA_FEED_LISTENER_STATE_NAME,
  L2_DATA_FEED_SERVICE_NAME,
  L2_LISTENER_DATA_FEED_HISTORY_QUEUE_NAME,
  L2_LISTENER_DATA_FEED_LATEST_QUEUE_NAME,
  L2_LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME,
  L2_WORKER_AGGREGATOR_QUEUE_NAME,
} from '../settings'
import { IAnswerUpdated, IDataFeedListenerWorkerL2, IListenerConfig } from '../types'
import { buildSubmissionRoundJobId } from '../utils'
import { listenerService } from './listener'
import { ProcessEventOutputType } from './types'

const FILE_NAME = import.meta.url
export async function buildListener(
  config: IListenerConfig[],
  redisClient: RedisClientType,
  logger: Logger,
) {
  const stateName = L2_DATA_FEED_LISTENER_STATE_NAME
  const service = L2_DATA_FEED_SERVICE_NAME
  const chain = CHAIN
  const eventName = 'AnswerUpdated'
  const latestQueueName = L2_LISTENER_DATA_FEED_LATEST_QUEUE_NAME
  const historyQueueName = L2_LISTENER_DATA_FEED_HISTORY_QUEUE_NAME
  const processEventQueueName = L2_LISTENER_DATA_FEED_PROCESS_EVENT_QUEUE_NAME
  const workerQueueName = L2_WORKER_AGGREGATOR_QUEUE_NAME
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
    logger,
  })
}

async function processEvent({ iface, logger }: { iface: ethers.utils.Interface; logger: Logger }) {
  const _logger = logger.child({ name: 'processEvent', file: FILE_NAME })

  async function wrapper(log): Promise<ProcessEventOutputType | undefined> {
    const eventData = iface.parseLog(log).args as unknown as IAnswerUpdated
    _logger.debug(eventData, 'eventData')

    const oracleAddress = log.address
    const roundId = eventData.roundId.toNumber()
    const jobName = 'event'

    const jobId = buildSubmissionRoundJobId({
      oracleAddress,
      roundId,
      deploymentName: DEPLOYMENT_NAME,
    })
    const jobData: IDataFeedListenerWorkerL2 = {
      oracleAddress,
      roundId,
      answer: eventData.current.toNumber(),
      workerSource: 'event',
    }
    _logger.debug(jobData, 'jobData')

    return { jobName, jobId, jobData, jobQueueSettings: AGGREGATOR_QUEUE_SETTINGS }
  }

  return wrapper
}
