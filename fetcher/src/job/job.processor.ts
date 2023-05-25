import { InjectQueue, Processor, WorkerHost } from '@nestjs/bullmq'
import { Logger } from '@nestjs/common'
import { Job, Queue } from 'bullmq'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { fetchData, aggregateData, shouldReport } from './job.utils'
import { insertMultipleData, insertAggregateData, fetchDataFeed } from './job.api'
import { DEVIATION_QUEUE_NAME, FETCHER_QUEUE_NAME, WORKER_OPTS } from '../settings'
import { IDeviationData } from './job.types'

@Processor(FETCHER_QUEUE_NAME, WORKER_OPTS)
export class JobProcessor extends WorkerHost {
  constructor(@InjectQueue(DEVIATION_QUEUE_NAME) private deviationQueue: Queue) {
    super()
  }
  private readonly logger = new Logger(JobProcessor.name)

  async process(job: Job<any, any, string>): Promise<any> {
    const inData = job.data
    const timestamp = new Date(Date.now()).toString()

    const keys = Object.keys(inData)
    if (keys.length == 0 || keys.length > 1) {
      throw new FetcherError(FetcherErrorCode.UnexpectedNumberOfJobs, String(keys.length))
    } else {
      const adapterHash = keys[0]
      const aggregatorId = inData[adapterHash].aggregatorId
      const feeds = inData[adapterHash].feeds
      const data = await fetchData(feeds, this.logger)
      const aggregate = aggregateData(data)
      const threshold = inData[adapterHash].threshold
      const absoluteThreshold = inData[adapterHash].absoluteThreshold

      const oracleAddress = inData[adapterHash].address
      const aggregatorHash = inData[adapterHash].aggregatorHash

      try {
        const { value: lastSubmission } = await fetchDataFeed({
          aggregatorHash,
          logger: this.logger
        })
        let response = await insertMultipleData({ aggregatorId, timestamp, data })
        this.logger.debug(response)
        response = await insertAggregateData({
          aggregatorId,
          timestamp,
          value: aggregate
        })
        this.logger.debug(response)
        const outData: IDeviationData = {
          timestamp: timestamp,
          submission: aggregate,
          oracleAddress
        }
        if (shouldReport(Number(lastSubmission), aggregate, 8, threshold, absoluteThreshold)) {
          this.deviationQueue.add('fetcher-submission', outData, {
            removeOnFail: true,
            removeOnComplete: true
          })
          this.logger.debug('added deviation queue', oracleAddress)
        }
      } catch (e) {
        this.logger.error(e)
      }
    }
  }
}
