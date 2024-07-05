import { InjectQueue, Processor, WorkerHost } from '@nestjs/bullmq'
import { Logger } from '@nestjs/common'
import { Job, Queue } from 'bullmq'
import { DEVIATION_QUEUE_NAME, FETCHER_QUEUE_NAME, WORKER_OPTS } from '../settings'
import { fetchDataFeedByAggregatorId, insertAggregateData, insertMultipleData } from './job.api'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { IDeviationData } from './job.types'
import { aggregateData, fetchData, shouldReport } from './job.utils'

@Processor(FETCHER_QUEUE_NAME, WORKER_OPTS)
export class JobProcessor extends WorkerHost {
  constructor(@InjectQueue(DEVIATION_QUEUE_NAME) private deviationQueue: Queue) {
    super()
  }
  private readonly logger = new Logger(JobProcessor.name)

  async process(job: Job<any, any, string>): Promise<any> {
    const inData = job.data
    const timestamp = new Date(Date.now()).toISOString()

    const keys = Object.keys(inData)
    if (keys.length == 0 || keys.length > 1) {
      throw new FetcherError(FetcherErrorCode.UnexpectedNumberOfJobs, String(keys.length))
    } else {
      const adapterHash = keys[0]
      const aggregatorId = inData[adapterHash].aggregatorId
      const feeds = inData[adapterHash].feeds
      const decimals = inData[adapterHash].decimals
      const data = await fetchData(feeds, decimals, this.logger)
      const aggregate = aggregateData(data)

      const threshold = inData[adapterHash].threshold
      const absoluteThreshold = inData[adapterHash].absoluteThreshold

      const oracleAddress = inData[adapterHash].address

      try {
        const { value: lastSubmission } = await fetchDataFeedByAggregatorId({
          aggregatorId,
          logger: this.logger,
        })
        let response = await insertMultipleData({ aggregatorId, timestamp, data })

        response = await insertAggregateData({
          aggregatorId,
          timestamp,
          value: aggregate,
        })

        const outData: IDeviationData = {
          timestamp: timestamp,
          submission: aggregate,
          oracleAddress,
        }
        if (
          shouldReport(Number(lastSubmission), aggregate, decimals, threshold, absoluteThreshold)
        ) {
          this.deviationQueue.add('fetcher-submission', outData, {
            removeOnFail: true,
            removeOnComplete: true,
          })
          this.logger.debug('added deviation queue', oracleAddress)
        }
      } catch (e) {
        this.logger.error(e)
      }
    }
  }
}
