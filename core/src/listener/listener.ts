import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { State } from './state'
import { IListenerConfig } from '../types'
import { BULLMQ_CONNECTION, REMOVE_ON_COMPLETE, REMOVE_ON_FAIL } from '../settings'
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
  const listenerRedisKey = `listener` // FIXME add unique name

  async function wrapper(job: Job) {
    const inData: ILatestListenerJob = job.data
    const { contractAddress } = inData

    let latestBlock: number
    let observedBlock: number

    try {
      latestBlock = await state.latestBlockNumber()
    } catch (e) {
      logger.error('latest block number failure')
      logger.error(e)
      // TODO skip
      // Pass to historical queue with higher priority
      throw e
    }

    try {
      observedBlock = Number(await redisClient.get(listenerRedisKey))
    } catch (e) {
      // TODO skip
      logger.error('observed block failure')
      logger.error(e)
      throw e
    }

    try {
      logger.info(`latestWorker ${observedBlock}-${latestBlock}`)

      if (latestBlock && latestBlock > observedBlock) {
        await redisClient.set(listenerRedisKey, latestBlock)

        const events = await state.queryEvent(contractAddress, observedBlock + 1, latestBlock)
        for (const event of events) {
          const outData: IProcessEventListenerJob = {
            contractAddress,
            event
          }
          // FIXME can we get the block number for jobId?
          await processEventQueue.add('latest', outData, {
            // removeOnComplete: REMOVE_ON_COMPLETE,
            // removeOnFail: REMOVE_ON_FAIL
            // attempts: 10,
            // backoff: 1_000
          })
        }
      }
    } catch (e) {
      // await redisClient.set(LISTENER_REDIS_KEY, observedBlock)

      for (let blockNumber = observedBlock + 1; blockNumber <= latestBlock; ++blockNumber) {
        const outData: IHistoryListenerJob = { contractAddress, blockNumber }
        await historyListenerQueue.add('rpc-failure', outData, {
          jobId: `${blockNumber.toString()}-${contractAddress}`,
          removeOnComplete: REMOVE_ON_COMPLETE,
          removeOnFail: REMOVE_ON_FAIL
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
