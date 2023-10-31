import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { Job, Queue, Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { getOperatorAddressL2 } from '../api'
import { OraklError, OraklErrorCode } from '../errors'
import {
  BULLMQ_CONNECTION,
  DATA_FEED_FULFILL_GAS_MINIMUM,
  DATA_FEED_WORKER_L2_STATE_NAME,
  DEPLOYMENT_NAME,
  HEARTBEAT_QUEUE_NAME,
  L2_CHAIN,
  L2_PROVIDER,
  REMOVE_ON_COMPLETE,
  REPORTER_AGGREGATOR_L2_QUEUE_NAME,
  SUBMIT_HEARTBEAT_QUEUE_NAME,
  WORKER_AGGREGATOR_L2_QUEUE_NAME
} from '../settings'
import { IDataFeedListenerWorkerL2, QueueType } from '../types'
import { buildSubmissionRoundJobId } from '../utils'
import { getAggregators, getL2AddressGivenL1Address } from './api'
import { buildTransaction, oracleRoundStateCall } from './data-feed.utils'
import { State } from './state'
import { watchman } from './watchman'

const FILE_NAME = import.meta.url

/**
 * Get all active aggregators, create their initial jobs, and submit
 * them to the [heartbeat] queue. Launch [aggregator] and [heartbeat]
 * workers.
 *
 * @param {RedisClientType} redis client
 * @param {Logger} pino logger
 */
export async function worker(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'worker', file: FILE_NAME })

  // Queues
  const heartbeatQueue = new Queue(HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  const submitHeartbeatQueue = new Queue(SUBMIT_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)
  const reporterQueue = new Queue(REPORTER_AGGREGATOR_L2_QUEUE_NAME, BULLMQ_CONNECTION)
  const state = new State({
    redisClient,
    stateName: DATA_FEED_WORKER_L2_STATE_NAME,
    heartbeatQueue,
    submitHeartbeatQueue,
    chain: L2_CHAIN,
    logger
  })
  await state.clear()

  const activeAggregators = await getAggregators({ chain: L2_CHAIN, active: true, logger })
  if (activeAggregators.length == 0) {
    logger.warn('No active aggregators')
  }

  // Launch all active aggregators
  for (const aggregator of activeAggregators) {
    logger.info('active aggregators')

    await state.add(aggregator.aggregatorHash)
  }

  // [aggregator] worker
  const aggregatorWorker = new Worker(
    WORKER_AGGREGATOR_L2_QUEUE_NAME,
    aggregatorJob(reporterQueue, _logger),
    {
      ...BULLMQ_CONNECTION
    }
  )
  aggregatorWorker.on('error', (e) => {
    logger.error(e)
    console.log(e)
  })

  const watchmanServer = await watchman({ state, logger })

  async function handleExit() {
    logger.info('Exiting. Wait for graceful shutdown.')
    await redisClient.quit()
    await aggregatorWorker.close()
    await watchmanServer.close()
  }
  process.on('SIGINT', handleExit)
  process.on('SIGTERM', handleExit)
}

/**
 * [aggregator] worker receives both [event] and [heartbeat]
 * jobs. {event} jobs are created by listener. {heartbeat} jobs are
 * either created during a launch of a worker, or inside of a reporter.
 *
 * Worker accepts job, parses the request, fetches the latest
 * aggregated data from the Orakl Network API for a specific
 * aggregator, and communicated with Aggregator smart contract to find
 * out the which round ID, it can submit the latest value. Then, it
 * create a new job and passes it to reporter worker.
 *
 * @param {QueueType} submit heartbeat queue
 * @param {QueueType} reporter queue
 * @param {Logger} pino logger
 * @return {} [aggregator] job processor
 */
export function aggregatorJob(reporterQueue: QueueType, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob' })
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job: Job) {
    const inData: IDataFeedListenerWorkerL2 = job.data
    logger.info(inData, 'inData')

    const { oracleAddress, workerSource } = inData
    try {
      // TODO store in ephemeral state
      const { l2AggregatorAddress } = await getL2AddressGivenL1Address({
        oracleAddress,
        chain: L2_CHAIN,
        logger
      })

      const answer = inData.answer
      console.log('answer', answer.toString())
      logger.debug({ oracleAddress, fetchedData: answer }, 'Latest data')
      const operatorAddress = await getOperatorAddressL2({
        oracleAddress: l2AggregatorAddress,
        logger
      })

      const oracleRoundState = await oracleRoundStateCall({
        oracleAddress: l2AggregatorAddress,
        operatorAddress,
        logger,
        provider: L2_PROVIDER
      })
      logger.debug(oracleRoundState, 'oracleRoundState')

      const { roundId, eligibleToSubmit } = oracleRoundState

      if (eligibleToSubmit) {
        const tx = buildTransaction({
          payloadParameters: {
            roundId,
            submission: BigInt(answer)
          },
          to: l2AggregatorAddress,
          gasMinimum: DATA_FEED_FULFILL_GAS_MINIMUM,
          iface,
          logger
        })
        logger.debug(tx, 'tx')

        await reporterQueue.add(workerSource, tx, {
          jobId: buildSubmissionRoundJobId({
            oracleAddress,
            roundId,
            deploymentName: DEPLOYMENT_NAME
          }),
          removeOnComplete: REMOVE_ON_COMPLETE,
          // Reporter job can fail, and should be either retried or
          // removed. We need to remove the job after repeated failure
          // to prevent deadlock when running with a single node operator.
          // After removing the job on failure, we can resubmit the job
          // with the same unique ID representing the submission for
          // specific aggregator on specific round.
          //
          // FIXME Rethink!
          removeOnFail: true
        })
        return tx
      } else {
        const msg = `Non-eligible to submit for oracle ${oracleAddress} with operator ${operatorAddress}`
        throw new OraklError(OraklErrorCode.NonEligibleToSubmit, msg)
      }
    } catch (e) {
      // `FailedToFetchFromDataFeed` exception can be raised from `prepareDataForReporter`.
      // `aggregatorJob` is being triggered by either `fixed` or `event` worker.
      // `event` job will not be resubmitted. `fixed` job might be
      // resubmitted, however due to the nature of fixed job cycle, the
      // resubmission might be delayed more than is acceptable. For this
      // reason jobs processed with `aggregatorJob` job must be retried with
      // appropriate logic.
      logger.error(e)
      throw e
    }
  }

  return wrapper
}
