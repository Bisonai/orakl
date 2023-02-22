import { Job, Worker, Queue } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { loadWalletParameters, sendTransaction, buildWallet } from './utils'
import {
  REPORTER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION,
  FIXED_HEARTBEAT_QUEUE_NAME
} from '../settings'
import { IAggregatorWorkerReporter, IAggregatorHeartbeatWorker } from '../types'
import { IcnError, IcnErrorCode } from '../errors'

const FILE_NAME = import.meta.url

export async function reporter(_logger: Logger) {
  _logger.debug({ name: 'reporter', file: FILE_NAME })

  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })

  new Worker(REPORTER_AGGREGATOR_QUEUE_NAME, await job(wallet, _logger), BULLMQ_CONNECTION)
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
      const gasLimit = 300_000 // FIXME move to settings outside of code
      // TODO retry when transaction failed
      await sendTransaction({ wallet, to: aggregatorAddress, payload, _logger, gasLimit })
    } catch (e) {
      logger.error(e)
    } finally {
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
      await heartbeatQueue.add('fixed-heartbeat', outData, {
        delay: inData.delay,
        removeOnComplete: true,
        removeOnFail: true,
        jobId: aggregatorAddress
      })
      logger.debug({ job: 'added', delay: inData.delay }, 'job-added')
    }
  }

  return wrapper
}
