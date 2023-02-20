import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { ISubmissionInfo } from './types'
import { loadWalletParameters, sendTransaction, buildWallet } from './utils'
import {
  REPORTER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION,
  PUBLIC_KEY as OPERATOR_ADDRESS,
  REDIS_HOST,
  REDIS_PORT,
  DEPLOYMENT_NAME,
  FIXED_HEARTBEAT_QUEUE_NAME,
  toSubmitRoundIdKey,
  submittedRoundIdKey,
  submitterKey,
  lastSubmissionTimeKey
} from '../settings'
import { IAggregatorWorkerReporter, IAggregatorHeartbeatWorker } from '../types'
import { createRedisClient } from '../utils'
import { oracleRoundStateCall } from '../worker/utils'
import { IcnError, IcnErrorCode } from '../errors'

const FILE_NAME = import.meta.url

export async function reporter(_logger: Logger) {
  _logger.debug({ name: 'reporter', file: FILE_NAME })

  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })
  const redisClient = await createRedisClient(REDIS_HOST, REDIS_PORT)

  const regex = new RegExp(`${DEPLOYMENT_NAME}$`)
  for await (const key of redisClient.scanIterator()) {
    if (regex.test(key)) {
      redisClient.del(key)
    }
  }

  const worker = new Worker(
    REPORTER_AGGREGATOR_QUEUE_NAME,
    await job(wallet, redisClient, _logger),
    BULLMQ_CONNECTION
  )
}

function job(wallet, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  const heartbeatQueue = new Queue(FIXED_HEARTBEAT_QUEUE_NAME, BULLMQ_CONNECTION)

  async function wrapper(job: Job) {
    const inData: IAggregatorWorkerReporter = job.data
    logger.debug(inData, 'inData')

    try {
      const aggregatorAddress = inData.callbackAddress

      const payload = iface.encodeFunctionData('submit', [inData.roundId, inData.submission])
      await sendTransaction({ wallet, to: aggregatorAddress, payload, _logger })

      const allDelayed = (await heartbeatQueue.getJobs(['delayed'])).filter(
        (job) => job.opts.jobId == aggregatorAddress
      )

      if (allDelayed.length > 1) {
        throw new IcnError(IcnErrorCode.UnexpectedNumberOfJobsInQueue)
      } else if (allDelayed.length == 1) {
        const delayedJob = allDelayed[0]
        delayedJob.remove()

        logger.debug({ job: 'deleted' }, 'job-deleted')
      }

      const outData: IAggregatorHeartbeatWorker = {
        aggregatorAddress
      }
      const fixedJob = await heartbeatQueue.add('fixed-heartbeat', outData, {
        delay: 15_000, // FIXME
        removeOnComplete: true,
        removeOnFail: true,
        jobId: aggregatorAddress
      })
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  return wrapper
}
