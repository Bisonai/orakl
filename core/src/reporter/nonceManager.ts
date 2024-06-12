import { Job } from 'bullmq'
import { Logger } from 'pino'
import { NONCE_MANAGER_JOB_SETTINGS } from '../settings'
import { ITransactionParameters, ITransactionParametersWithNonce, QueueType } from '../types'
import { State } from './state'

export function nonceManager(
  reporterQueue: QueueType,
  jobName: string,
  state: State,
  logger: Logger
) {
  async function wrapper(job: Job) {
    const tx: ITransactionParameters = job.data
    const { to } = tx

    try {
      const nonce = await state.getAndIncrementNonce(to)
      const txWithNonce: ITransactionParametersWithNonce = { ...tx, nonce }
      await reporterQueue.add(jobName, txWithNonce, {
        jobId: job.id,
        ...NONCE_MANAGER_JOB_SETTINGS
      })
    } catch (e) {
      logger.error(
        e,
        `Failed to get and increment nonce for oracle with address ${to}. Retrying...`
      )
      throw e
    }
  }

  logger.debug('Nonce Manager job build')
  return wrapper
}
