import { Processor, WorkerHost } from '@nestjs/bullmq'
import { Logger } from '@nestjs/common'
import { Job } from 'bullmq'
import { FetcherError, FetcherErrorCode } from './job.errors'
import { fetchData, aggregateData } from './job.utils'
import { insertMultipleData, insertAggregateData } from './job.api'

@Processor('orakl-fetcher-queue')
export class JobProcessor extends WorkerHost {
  private readonly logger = new Logger(JobProcessor.name)

  async process(job: Job<any, any, string>): Promise<any> {
    const inData = job.data
    const timestamp = new Date(Date.now()).toString()

    const keys = Object.keys(inData)
    if (keys.length == 0 || keys.length > 1) {
      throw new FetcherError(FetcherErrorCode.UnexpectedNumberOfJobs, String(keys.length))
    } else {
      const aggregatorHash = keys[0]
      const aggregatorId = inData[aggregatorHash].aggregatorId
      const feeds = inData[aggregatorHash].feeds
      const data = await fetchData(feeds)
      const aggregate = aggregateData(data)

      try {
        let response = await insertMultipleData({ aggregatorId, timestamp, data })
        this.logger.debug(response)
        response = await insertAggregateData({ aggregatorId, timestamp, value: aggregate })
        this.logger.debug(response)
      } catch (e) {
        this.logger.error(e)
      }
    }
  }
}
