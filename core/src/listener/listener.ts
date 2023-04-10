import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { State } from './state'
import { IListenerConfig } from '../types'
import { BULLMQ_CONNECTION, LISTENER_JOB_SETTINGS, getObservedBlockRedisKey } from '../settings'
import {
  IProcessEventListenerJob,
  IHistoryListenerJob,
  ILatestListenerJob,
  ListenerInitType,
  ProcessEventOutputType
} from './types'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

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
 * @params {ListenerInitType} listener initialization type
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
  processEventQueueName,
  workerQueueName,
  processFn,
  redisClient,
  listenerInitType,
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
  workerQueueName: string
  processFn: (log: ethers.Event) => Promise<ProcessEventOutputType>
  redisClient: RedisClientType
  listenerInitType: ListenerInitType
  logger: Logger
}) {
  const latestListenerQueue = new Queue(latestQueueName, BULLMQ_CONNECTION)
  const historyListenerQueue = new Queue(historyQueueName, BULLMQ_CONNECTION)
  const processEventQueue = new Queue(processEventQueueName, BULLMQ_CONNECTION)
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
    listenerInitType,
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
    processEventJob({ workerQueue, processFn, logger }),
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

    const logPrefix = generateListenerLogPrefix(contractAddress, observedBlock, latestBlock)
    try {
      if (latestBlock > observedBlock) {
        await redisClient.set(observedBlockRedisKey, latestBlock)

        // The `observedBlock` block number is already processed,
        // therefore we do not need to re-query the same event in such
        // block again.
        const events = await state.queryEvent(contractAddress, observedBlock + 1, latestBlock)
        for (const [index, event] of events.entries()) {
          const outData: IProcessEventListenerJob = {
            contractAddress,
            event
          }
          const jobId = getUniqueEventIdentifier(event, index)
          await processEventQueue.add('latest', outData, {
            jobId,
            ...LISTENER_JOB_SETTINGS
          })
        }
        logger.info(logPrefix)
      } else {
        logger.info(`${logPrefix} noop`)
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
    const logPrefix = generateListenerLogPrefix(contractAddress, blockNumber, blockNumber)

    let events: ethers.Event[] = []
    try {
      events = await state.queryEvent(contractAddress, blockNumber, blockNumber)
    } catch (e) {
      logger.error(`${logPrefix} hist fail`)
      throw e
    }

    logger.info(`${logPrefix} hist`)

    for (const [index, event] of events.entries()) {
      const outData: IProcessEventListenerJob = {
        contractAddress,
        event
      }
      const jobId = getUniqueEventIdentifier(event, index)
      await processEventQueue.add('history', outData, {
        jobId,
        ...LISTENER_JOB_SETTINGS
      })
    }
  }

  return wrapper
}

/**
 * The [processEvent] listener worker accepts jobs from [processEvent]
 * queue. The jobs are submitted either by the [latest] or [history]
 * listener worker.
 *
 * @param {} function that processes event caught by listener
 */
function processEventJob({
  workerQueue,
  processFn,
  logger
}: {
  workerQueue: Queue
  processFn: (log: ethers.Event) => Promise<ProcessEventOutputType>
  logger: Logger
}) {
  const _logger = logger.child({ name: 'processEventJob', file: FILE_NAME })

  async function wrapper(job: Job) {
    const inData: IProcessEventListenerJob = job.data
    const { event } = inData
    _logger.debug(event, 'event')

    try {
      const { jobId, jobName, jobData, jobQueueSettings } = await processFn(event)
      if (jobData) {
        const queueSettings = jobQueueSettings ? jobQueueSettings : LISTENER_JOB_SETTINGS
        await workerQueue.add(jobName, jobData, {
          jobId,
          ...queueSettings
        })
        _logger.debug(`Listener submitted job [${jobId}] for [${jobName}]`)
      }
    } catch (e) {
      _logger.error(e, 'Error in user defined listener processing function')
      throw e
    }
  }

  return wrapper
}

/**
 * Auxiliary function to create a unique identifier for a give `event`
 * and `index` of the even within the transaction.
 *
 * @param {ethers.Event} event
 * @param {number} index of event within a transaction
 */
function getUniqueEventIdentifier(event: ethers.Event, index: number) {
  return `${event.blockNumber}-${event.transactionHash}-${index}`
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
