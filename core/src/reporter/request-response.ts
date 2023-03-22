import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { RequestResponseCoordinator__factory } from '@bisonai/orakl-contracts'
import { sendTransaction, loadWalletParameters, buildWallet } from './utils'
import { REPORTER_REQUEST_RESPONSE_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IRequestResponseWorkerReporter, RequestCommitmentRequestResponse } from '../types'

const FILE_NAME = import.meta.url

export async function reporter(redisClient: RedisClientType, _logger: Logger) {
  _logger.debug({ name: 'reporter', file: FILE_NAME })
  const { privateKey, providerUrl } = loadWalletParameters()
  const wallet = await buildWallet({ privateKey, providerUrl })
  new Worker(REPORTER_REQUEST_RESPONSE_QUEUE_NAME, await job(wallet, _logger), BULLMQ_CONNECTION)
}

function job(wallet, _logger: Logger) {
  const logger = _logger.child({ name: 'job', file: FILE_NAME })
  const iface = new ethers.utils.Interface(RequestResponseCoordinator__factory.abi)

  async function wrapper(job) {
    const inData: IRequestResponseWorkerReporter = job.data
    logger.debug(inData, 'inData')

    try {
      const data = typeof inData.data === 'number' ? Math.floor(inData.data) : inData.data
      logger.debug(data, 'data')

      const rc: RequestCommitmentRequestResponse = [
        inData.blockNum,
        inData.accId,
        inData.callbackGasLimit,
        inData.sender
      ]

      logger.debug(inData.requestId, 'inData.requestId')
      logger.debug(rc, 'rc')
      logger.debug(data, 'data')
      logger.debug(inData.isDirectPayment, 'inData.isDirectPayment')

      const payload = iface.encodeFunctionData('fulfillDataRequest', [
        inData.requestId,
        data,
        rc,
        inData.isDirectPayment
      ])

      await sendTransaction({ wallet, to: inData.callbackAddress, payload, _logger })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}
