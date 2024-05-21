import { Job, Queue, Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { BULLMQ_CONNECTION, getObservedBlockRedisKey, LISTENER_JOB_SETTINGS } from '../settings'
import { IListenerConfig } from '../types'
import { upsertListenerObservedBlock } from './api'
import { State } from './state'
import { IHistoryListenerJob, ILatestListenerJob, ProcessEventOutputType } from './types'
import { watchman } from './watchman'

/**
 * The listener service is used for tracking events emmitted by smart
 * contracts. Tracked events are subsequently send to BullMQ queue to
 * be processed by follow-up service. The listener service guarantees
 * that no event is missed.
 *
 * The listener service can be controlled through changes to its
 * ephemeral state. It keeps information about currently tracked
 * events, and allows to activate/deactivate events while the listener
 * service is running. The listener's ephemeral state is updated
 * through Watchman REST API service.
 *
 * @param {IListenerConfig[]} listener definition
 * @param {ethers.ContractInterface} event ABI
 * @param {string} state name
 * @param {string} service name
 * @param {string} chain name
 * @param {string} event name
 * @param {string} name of [latest] queue
 * @param {string} name of [history] queue
 * @param {string} name of [processEvent] queue
 * @param {string} name of [worker] queue
 * @param {(log: ethers.Event) => Promise<ProcessEventOutputType>} event processing function
 * @param {RedisClientType} redis client
 * @param {Logger} pino logger
 */
export async function listenerService({
  config,
  abi,
  stateName,
  service,
  chain,
  eventName,
  latestQueueName,
  historyQueueName,
  workerQueueName,
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
  workerQueueName: string
  processFn: (log: ethers.Event) => Promise<ProcessEventOutputType | undefined>
  redisClient: RedisClientType
  logger: Logger
}) {
  const latestListenerQueue = new Queue(latestQueueName, BULLMQ_CONNECTION)
  const historyListenerQueue = new Queue(historyQueueName, BULLMQ_CONNECTION)
  const workerQueue = new Queue(workerQueueName, BULLMQ_CONNECTION)

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
      workerQueue,
      redisClient,
      processFn,
      logger
    }),
    BULLMQ_CONNECTION
  )
  latestWorker.on('error', (e) => {
    logger.error(e)
  })

  const historyWorker = new Worker(
    historyQueueName,
    historyJob({ state, logger, processFn, redisClient, workerQueue }),
    BULLMQ_CONNECTION
  )
  historyWorker.on('error', (e) => {
    logger.error(e)
  })

  for (const listener of config) {
    await state.add(listener.id)
  }

  const watchmanServer = await watchman({ state, logger })

  async function handleExit() {
    logger.debug('Exiting. Wait for graceful shutdown.')

    await latestWorker.close()
    await historyWorker.close()
    await state.clear()
    watchmanServer.close()
    await redisClient.quit()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}

/**
 * The responsibility of the [latest] listener worker is to stay
 * up-to-date with the most recent blocks and scan them for events
 * emitted by smart contracts defined within Orakl Network permanent
 * state. The [latest] listener worker operates on one or more blocks
 * at the time. The [latest] job is launched as a repeatable job from
 * within an ephemeral state's auxiliary function `add(id: string)`
 * where `id` represents unique identifier of listener defined in the
 * Orakl Network permanent state. Similarly, the [latest] job can be
 * deactived through ephemeral's state function `remove(id: string)`.
 *
 * When the event query fails, the query job is sent to [history] listener
 * worker through [history] queue where the event query is
 * retried. Successfully queried events are send to [processEvent] listener
 * worker through [processEvent] queue for further processing.
 *
 * @param {State} ephemeral state of listener
 * @param {Queue} queue that accepts jobs to process caught events
 * @param {Queue} queue that accepts jobs to retry failed event queries
 * @param {RedisClientType} redist client
 * @param {Logger} pino logger
 */
function latestJob({
  state,
  workerQueue,
  historyListenerQueue,
  redisClient,
  processFn,
  logger
}: {
  state: State
  historyListenerQueue: Queue
  redisClient: RedisClientType
  workerQueue: Queue
  processFn: (log: ethers.Event) => Promise<ProcessEventOutputType | undefined>
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
      // We assume that redis cache has been initialized within
      // `State.add` method call.
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

    if (latestBlock < observedBlock) {
      logger.warn('latestBlock < observedBlock. Updating observed block to revert the condition.')
      observedBlock = Math.max(0, latestBlock - 1)
    }

    const logPrefix = generateListenerLogPrefix(contractAddress, observedBlock, latestBlock)
    try {
      if (latestBlock > observedBlock) {
        const lockObservedBlock = observedBlock + 1
        // The `observedBlock` block number is already processed,
        // therefore we do not need to re-query the same event in such
        // block again.
        for (let blockNumber = lockObservedBlock; blockNumber <= latestBlock; ++blockNumber) {
          const events = await state.queryEvent(contractAddress, blockNumber, blockNumber)
          for (const [_, event] of events.entries()) {
            const jobMetadata = await processFn(event)
            if (jobMetadata) {
              const { jobId, jobName, jobData, jobQueueSettings } = jobMetadata
              const queueSettings = jobQueueSettings ? jobQueueSettings : LISTENER_JOB_SETTINGS
              await workerQueue.add(jobName, jobData, {
                jobId,
                ...queueSettings
              })
              logger.debug(`Listener submitted job [${jobId}] for [${jobName}]`)
            }
          }
          await upsertListenerObservedBlock({
            blockKey: observedBlockRedisKey,
            blockNumber,
            logger: this.logger
          })
          await redisClient.set(observedBlockRedisKey, blockNumber)
          observedBlock += 1 // in case of failure, dont add processed blocks to history queue
        }
        logger.debug(logPrefix)
      } else {
        logger.debug(`${logPrefix} noop`)
      }
    } catch (e) {
      // Querying the latest events or passing data to [process] queue
      // failed. Repeateable [latest] job will continue listening for
      // new blocks, and the blocks which failed to be scanned for
      // events will be retried through [history] job.
      logger.warn(`${logPrefix} fail`)

      for (let blockNumber = observedBlock + 1; blockNumber <= latestBlock; ++blockNumber) {
        const outData: IHistoryListenerJob = { contractAddress, blockNumber }
        await historyListenerQueue.add('failure', outData, {
          jobId: `${blockNumber.toString()}-${contractAddress}`,
          ...LISTENER_JOB_SETTINGS
        })
      }
    }
  }

  return wrapper
}

/**
 * The [history] listener worker processes all jobs from the [history]
 * queue. Jobs in history queue are inserted either during the launch
 * of the listener (all unobserved blocks), or when event query fails
 * within the [latest] listener worker. Unlike the [latest] listener
 * worker, the [history] listener worker operates always on single
 * block at the time.
 *
 * Successfully queried events are send to [processEvent] listener
 * worker through [processEvent] queue for further processing.
 *
 * @param {State} ephemeral state of listener
 * @param {Queue} queue that accepts jobs to process caught events
 * @param {Logger} pino logger
 */
function historyJob({
  state,
  logger,
  processFn,
  redisClient,
  workerQueue
}: {
  state: State
  logger: Logger
  redisClient: RedisClientType
  workerQueue: Queue
  processFn: (log: ethers.Event) => Promise<ProcessEventOutputType | undefined>
}) {
  async function wrapper(job: Job) {
    const inData: IHistoryListenerJob = job.data
    const { contractAddress, blockNumber } = inData
    const logPrefix = generateListenerLogPrefix(contractAddress, blockNumber, blockNumber)

    try {
      const observedBlockRedisKey = getObservedBlockRedisKey(contractAddress)
      const observedBlock = Number(await redisClient.get(observedBlockRedisKey))
      const events = await state.queryEvent(contractAddress, blockNumber, blockNumber)
      for (const [_, event] of events.entries()) {
        const jobMetadata = await processFn(event)
        if (jobMetadata) {
          const { jobId, jobName, jobData, jobQueueSettings } = jobMetadata
          const queueSettings = jobQueueSettings ? jobQueueSettings : LISTENER_JOB_SETTINGS
          await workerQueue.add(jobName, jobData, {
            jobId,
            ...queueSettings
          })
          logger.debug(`Listener submitted job [${jobId}] for [${jobName}]`)
        }
      }

      if (blockNumber > observedBlock) {
        await upsertListenerObservedBlock({
          blockKey: observedBlockRedisKey,
          blockNumber,
          logger
        })
        await redisClient.set(observedBlockRedisKey, blockNumber)
      }
    } catch (e) {
      logger.error(`${logPrefix} hist fail`)
      throw e
    }
  }

  return wrapper
}

/**
 * Auxiliary function that generate a consisten log prefix, that is
 * used both by the [latest] and [history] listener worker.
 *
 * @param {string} contractAddress
 * @param {number} start block number
 * @param {number} end block number
 */
function generateListenerLogPrefix(contractAddress: string, fromBlock: number, toBlock: number) {
  return `${contractAddress} ${fromBlock}-${toBlock} (${toBlock - fromBlock})`
}
