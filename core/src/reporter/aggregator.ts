import { Worker } from 'bullmq'
import { ethers } from 'ethers'
import { Logger } from 'pino'
import { Aggregator__factory } from '@bisonai-cic/icn-contracts'
import { sendTransaction, buildWallet } from './utils'
import { REPORTER_AGGREGATOR_QUEUE_NAME, BULLMQ_CONNECTION } from '../settings'
import { IAggregatorWorkerReporter } from '../types'

const FILE_NAME = import.meta.url

export async function aggregatorReporter(_logger: Logger) {
  _logger.debug({ name: 'aggregatorReporter', file: FILE_NAME })

  const wallet = buildWallet(_logger)
  new Worker(
    REPORTER_AGGREGATOR_QUEUE_NAME,
    await aggregatorJob(wallet, _logger),
    BULLMQ_CONNECTION
  )
}

function aggregatorJob(wallet, _logger: Logger) {
  const logger = _logger.child({ name: 'aggregatorJob', file: FILE_NAME })
  const iface = new ethers.utils.Interface(Aggregator__factory.abi)

  async function wrapper(job) {
    const inData: IAggregatorWorkerReporter = job.data
    logger.debug(inData, 'inData')

    try {
      const payload = iface.encodeFunctionData('submit', [inData.roundId, inData.submission])
      await sendTransaction({ wallet, to: inData.callbackAddress, payload, _logger })
    } catch (e) {
      logger.error(e)
    }
  }

  return wrapper
}
