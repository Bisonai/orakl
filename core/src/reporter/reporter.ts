import { Job } from 'bullmq'
import { Logger } from 'pino'
import {
  sendTransaction,
  sendTransactionDelegatedFee,
  sendTransactionCaver,
  makeSubmissionData,
  isSubmitMethod
} from './utils'
import { State } from './state'
import { ITransactionParameters } from '../types'
import { OraklError, OraklErrorCode } from '../errors'
import { storeSubmission } from '../reporter/api'
import { ISubmissionData } from './types'

export function reporter(state: State, logger: Logger) {
  async function wrapper(job: Job) {
    const inData: ITransactionParameters = job.data
    logger.debug(inData, 'inData')

    const { payload, gasLimit, to } = inData

    const wallet = state.wallets[to]
    if (!wallet) {
      const msg = `Wallet for oracle ${to} is not active`
      logger.error(msg)
      throw new OraklError(OraklErrorCode.WalletNotActive, msg)
    }

    let delegatorOkay = true
    const NUM_TRANSACTION_TRIALS = 3
    const txParams = { wallet, to, payload, gasLimit, logger }

    for (let i = 0; i < NUM_TRANSACTION_TRIALS; ++i) {
      if (state.delegatedFee && delegatorOkay) {
        try {
          await sendTransactionDelegatedFee(txParams)
          break
        } catch (e) {
          if (e.code == OraklErrorCode.DelegatorServerIssue) {
            delegatorOkay = false
          }
        }
      } else if (state.delegatedFee) {
        try {
          await sendTransactionCaver(txParams)
          break
        } catch (e) {
          if (![OraklErrorCode.CaverTxTransactionFailed].includes(e.code)) {
            throw e
          }
        }
      } else {
        try {
          await sendTransaction(txParams)
          break
        } catch (e) {
          if (
            ![
              OraklErrorCode.TxNotMined,
              OraklErrorCode.TxProcessingResponseError,
              OraklErrorCode.TxMissingResponseError
            ].includes(e.code)
          ) {
            throw e
          }

          logger.info(`Retrying transaction. Trial number: ${i}`)
        }
      }
    }

    // check if payload uses Submit method, which indicates submission if for Aggregator Price Feeds
    // Needs to store all the aggregator submissions on db.
    if (isSubmitMethod(wallet, payload)) {
      const submissionData: ISubmissionData = await makeSubmissionData({
        to,
        payload,
        logger
      })
      try {
        const response = await storeSubmission({ submissionData, logger })
        logger.info(`Submission is stored.`, response.data)
      } catch (e) {
        logger.error('Storing Submission data failed.')
        logger.error('submissionData:', submissionData)
        throw e
      }
    }
  }

  logger.debug('Reporter job built')
  return wrapper
}
