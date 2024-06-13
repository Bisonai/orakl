import { NonceManager } from '@ethersproject/experimental'
import { Job } from 'bullmq'
import { Logger } from 'pino'
import { OraklError, OraklErrorCode } from '../errors'
import { ITransactionParametersWithNonce } from '../types'
import { State } from './state'
import {
  CaverWallet,
  sendTransaction,
  sendTransactionCaver,
  sendTransactionDelegatedFee
} from './utils'

export function reporter(state: State, logger: Logger) {
  async function wrapper(job: Job) {
    const inData: ITransactionParametersWithNonce = job.data
    logger.debug(inData, 'inData')

    const { payload, gasLimit, to, nonce } = inData

    const wallet = state.wallets[to]
    if (!wallet) {
      const msg = `Wallet for oracle ${to} is not active`
      logger.error(msg)
      throw new OraklError(OraklErrorCode.WalletNotActive, msg)
    }

    let delegatorOkay = true
    const NUM_TRANSACTION_TRIALS = 3
    const txParams = { to, payload, gasLimit, logger, nonce }

    for (let i = 0; i < NUM_TRANSACTION_TRIALS; ++i) {
      if (state.delegatedFee && delegatorOkay) {
        try {
          await sendTransactionDelegatedFee({ ...txParams, wallet: wallet as CaverWallet })
          break
        } catch (e) {
          delegatorOkay = false
        }
      } else if (state.delegatedFee) {
        try {
          await sendTransactionCaver({ ...txParams, wallet: wallet as CaverWallet })
          break
        } catch (e) {
          if (![OraklErrorCode.CaverTxTransactionFailed].includes(e.code)) {
            throw e
          }
        }
      } else {
        try {
          await sendTransaction({ ...txParams, wallet: wallet as NonceManager })
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
  }

  logger.debug('Reporter job built')
  return wrapper
}
