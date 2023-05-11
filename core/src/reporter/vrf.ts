import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { loadWalletParameters, sendTransaction, buildWallet } from './utils'
import { REPORTER_VRF_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { ITransactionParameters } from '../types'

const FILE_NAME = import.meta.url

export async function reporter(redisClient: RedisClientType, _logger: Logger) {
  const logger = _logger.child({ name: 'reporter', file: FILE_NAME })

  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })
  new Worker(REPORTER_VRF_QUEUE_NAME, await job(wallet, logger), BULLMQ_CONNECTION)
}

function job(wallet, logger: Logger) {
  async function wrapper(job) {
    const inData: ITransactionParameters = job.data
    logger.debug(inData, 'inData')

    try {
      const payload = inData.payload
      const gasLimit = inData.gasLimit
      const to = inData.to
      await sendTransaction({ wallet, to, payload, gasLimit, logger })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}
