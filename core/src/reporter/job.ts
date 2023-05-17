import { Logger } from 'pino'
import { ITransactionParameters } from '../types'
import { sendTransaction } from './utils'

export function job(wallet, logger: Logger) {
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
