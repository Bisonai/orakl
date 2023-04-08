import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { State } from './state'
import { IListenerConfig } from '../types'
import {
  BULLMQ_CONNECTION,
  REMOVE_ON_COMPLETE,
  REMOVE_ON_FAIL,
  getObservedBlockRedisKey
} from '../settings'
import { IProcessEventListenerJob, IHistoryListenerJob, ILatestListenerJob } from './types'
import { watchman } from './watchman'

/**
 * Listener scans the latest, but also historical blocks to find out
 * if there is a event emitted from a set of smart contracts.
 */
export async function listener({
  config,
  abi,
  stateName,
  service,
  chain,
  eventName,
  latestQueueName,
  historyQueueName,
  processEventQueueName,
  processFn,
  redisClient,
  logger
}: {
  config: IListenerConfig[]
  abi: ethers.ContractInterface
  stateName: string
  service: string
  chain: string
  eventName: string
  latestQueueName: string
  historyQueueName: string
  processEventQueueName: string
  processFn // FIXME
  redisClient: RedisClientType
  logger: Logger
}) {
  const latestListenerQueue = new Queue(latestQueueName, BULLMQ_CONNECTION)
  const historyListenerQueue = new Queue(historyQueueName, BULLMQ_CONNECTION)
  const processEventQueue = new Queue(processEventQueueName, BULLMQ_CONNECTION)

  const state = new State({
    redisClient,
    latestListenerQueue,
    historyListenerQueue,
    stateName,
    service,
    chain,
    eventName,
    abi,
    logger
  })
  await state.clear()

  const latestWorker = new Worker(
    latestQueueName,
    latestJob({
      state,
      historyListenerQueue,
      processEventQueue,
      redisClient,
      logger
    }),
    BULLMQ_CONNECTION
  )
  latestWorker.on('error', (e) => {
    logger.error(e)
  })

  const historyWorker = new Worker(
    historyQueueName,
    historyJob({ state, processEventQueue, logger }),
    BULLMQ_CONNECTION
  )
  historyWorker.on('error', (e) => {
    logger.error(e)
  })

  const processEventWorker = new Worker(
    processEventQueueName,
    processEventJob(processFn),
    BULLMQ_CONNECTION
  )
  processEventWorker.on('error', (e) => {
    logger.error(e)
  })

  for (const listener of config) {
    await state.add(listener.id)
  }

  const watchmanServer = await watchman({ state, logger })

  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')

    await latestWorker.close()
    await historyWorker.close()
    await processEventWorker.close()
    await state.clear()
    await watchmanServer.close()
    await redisClient.quit()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}

function latestJob({
  state,
  processEventQueue,
  historyListenerQueue,
  redisClient,
  logger
}: {
  state: State
  processEventQueue: Queue
  historyListenerQueue: Queue
  redisClient: RedisClientType
  logger: Logger
}) {
  async function wrapper(job: Job) {
    const inData: ILatestListenerJob = job.data
    const { contractAddress } = inData
    const observedBlockRedisKey = getObservedBlockRedisKey(contractAddress)

    let latestBlock: number
    let observedBlock: number

    try {
      latestBlock = await state.latestBlockNumber()
    } catch (e) {
      // The observed block number has not been updated, therefore
      // we do not need to submit job to history queue. The next
      // repeatable job will re-request the latest block number and
      // continue from there.
      logger.error('Failed to fetch the latest block number.')
      logger.error(e)
      throw e
    }

    try {
      // FIXME failure of missing the value inside of redis cache
      observedBlock = Number(await redisClient.get(observedBlockRedisKey))
    } catch (e) {
      // Similarly to the failure during fetching the latest block
      // number, this error doesn't require job resubmission. The next
      // repeatable job will re-request the latest observed block number and
      // continue from there.
      logger.error('Failed to fetch the latest observed block from Redis.')
      logger.error(e)
      throw e
    }

    try {
      logger.info(
        `${contractAddress} ${observedBlock}-${latestBlock} (${latestBlock - observedBlock})`
      )

      if (latestBlock > observedBlock) {
        await redisClient.set(observedBlockRedisKey, latestBlock)

        const events = await state.queryEvent(contractAddress, observedBlock + 1, latestBlock)
        for (const event of events) {
          const outData: IProcessEventListenerJob = {
            contractAddress,
            event
          }
          // TODO can we get the block number for jobId?
          await processEventQueue.add('latest', outData, {
            // removeOnComplete: REMOVE_ON_COMPLETE,
            // removeOnFail: REMOVE_ON_FAIL
            // attempts: 10,
            // backoff: 1_000
          })
        }
      } else {
        logger.info(
          `${contractAddress} ${observedBlock}-${latestBlock} (${latestBlock - observedBlock}) noop`
        )
      }
    } catch (e) {
      logger.warn(
        `${contractAddress} ${observedBlock}-${latestBlock} (${latestBlock - observedBlock}) failed`
      )
      // Querying the latest events or passing data to [process] queue
      // failed. Repeateable [latest] job will continue listening for
      // new blocks, and the blocks which failed to be scanned for
      // events will be retried through [history] job.
      for (let blockNumber = observedBlock + 1; blockNumber <= latestBlock; ++blockNumber) {
        const outData: IHistoryListenerJob = { contractAddress, blockNumber }
        await historyListenerQueue.add('failure', outData, {
          jobId: `${blockNumber.toString()}-${contractAddress}`
          // removeOnComplete: REMOVE_ON_COMPLETE,
          // removeOnFail: REMOVE_ON_FAIL
          // attempts: 10,
          // backoff: 1_000
        })
      }
    }
  }

  return wrapper
}

/**
 * [history] worker processes all jobs from [history] queue. Jobs in
 * history queue are either inserted during launch of the listener
 * (all unobserved blocks) or from failed queries of the [latest] listener.
 */
function historyJob({
  state,
  processEventQueue,
  logger
}: {
  state: State
  processEventQueue: Queue
  logger: Logger
}) {
  async function wrapper(job: Job) {
    const inData: IHistoryListenerJob = job.data
    const { contractAddress, blockNumber } = inData
    logger.info(`historyWorker ${blockNumber}`)

    const events = await state.queryEvent(contractAddress, blockNumber, blockNumber)
    for (const event of events) {
      const outData: IProcessEventListenerJob = {
        contractAddress,
        event
      }
      await processEventQueue.add('history', outData, {
        // removeOnComplete: REMOVE_ON_COMPLETE,
        // removeOnFail: REMOVE_ON_FAIL
        // attempts: 10,
        // backoff: 1_000
      })
    }
  }

  return wrapper
}

function processEventJob(processEventFn /* FIXME data type */) {
  async function wrapper(job: Job) {
    const inData: IProcessEventListenerJob = job.data
    const { event } = inData

    // TODO return data and jobId and submit the job from here
    processEventFn(event)
  }

  return wrapper
}
