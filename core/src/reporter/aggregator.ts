import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { RedisClientType } from 'redis'
import { Aggregator__factory } from '@bisonai/orakl-contracts'
import { ISubmissionInfo } from './types'
import { loadWalletParameters, sendTransaction, buildWallet, createRedisClient } from './utils'
import {
  REPORTER_AGGREGATOR_QUEUE_NAME,
  BULLMQ_CONNECTION,
  PUBLIC_KEY as OPERATOR_ADDRESS,
  REDIS_HOST,
  REDIS_PORT,
  DEPLOYMENT_NAME
} from '../settings'
import { IAggregatorWorkerReporter } from '../types'
import { oracleRoundStateCall } from '../worker/utils'

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

  new Worker(
    REPORTER_AGGREGATOR_QUEUE_NAME,
    await job(wallet, redisClient, _logger),
    BULLMQ_CONNECTION
  )
}

function job(wallet, redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job) {
    const inData: IAggregatorWorkerReporter = job.data
    logger.debug(inData, 'inData')

    try {
      const aggregatorAddress = inData.callbackAddress
      const roundId = inData.roundId
      const workerSource = inData.workerSource

      const oracleRoundState = await oracleRoundStateCall({
        aggregatorAddress,
        operatorAddress: OPERATOR_ADDRESS,
        roundId,
        logger
      })
      const subInfo = await getSubmissionInfo(redisClient, aggregatorAddress)

      if (
        oracleRoundState._eligibleToSubmit && // (Submission must be eligible) AND
        ((subInfo.toSubmitRoundId < inData.roundId && // ((Nobody is trying to submit the same round ID) AND
          subInfo.submittedRoundId < inData.roundId) || // (Nobody has finalized the submission)) OR
          (subInfo.submitter == workerSource && // ((I was the first submitter) AND
            subInfo.toSubmitRoundId == inData.roundId && // (I have already tried to submit) AND(=HOWEVER_
            subInfo.submittedRoundId < inData.roundId)) // (I have not succeeded yet))
      ) {
        await redisClient.set(toSubmitRoundIdKey(aggregatorAddress), roundId)
        await redisClient.set(submitterKey(aggregatorAddress), workerSource)

        const payload = iface.encodeFunctionData('submit', [inData.roundId, inData.submission])
        await sendTransaction({ wallet, to: aggregatorAddress, payload, _logger })

        await redisClient.set(submittedRoundIdKey(aggregatorAddress), inData.roundId)
      } else {
        logger.info(`Data for ${inData.roundId} has already been submitted!`)
      }
    } catch (e) {
      logger.error(e)
      throw e
    }
  }

  return wrapper
}

function toSubmitRoundIdKey(aggregatorAddress: string): string {
  return `${aggregatorAddress}-toSubmitRoundId-${DEPLOYMENT_NAME}`
}

function submittedRoundIdKey(aggregatorAddress: string): string {
  return `${aggregatorAddress}-submittedRoundId-${DEPLOYMENT_NAME}`
}

function submitterKey(aggregatorAddress: string): string {
  return `${aggregatorAddress}-submitter-${DEPLOYMENT_NAME}`
}

async function getSubmissionInfo(
  client: RedisClientType,
  aggregatorAddress: string
): Promise<ISubmissionInfo> {
  const [toSubmitRoundId, submittedRoundId, submitter] = await client.MGET([
    toSubmitRoundIdKey(aggregatorAddress),
    submittedRoundIdKey(aggregatorAddress),
    submitterKey(aggregatorAddress)
  ])

  return {
    toSubmitRoundId: Number(toSubmitRoundId),
    submittedRoundId: Number(submittedRoundId),
    submitter
  }
}
