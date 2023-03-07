import { Controller, Get, Body, Param, HttpStatus, HttpException } from '@nestjs/common'
import { JobDto } from './dto/job.dto'
import { InjectQueue } from '@nestjs/bullmq'
import { Queue } from 'bullmq'
import { Logger } from '@nestjs/common'
import { loadAggregator, extractFeeds } from './job.utils'

@Controller({
  version: '1'
})
export class JobController {
  private readonly logger = new Logger(JobController.name)

  constructor(@InjectQueue('orakl-fetcher-queue') private queue: Queue) {}

  @Get('start/:aggregator')
  async start(@Param('aggregator') aggregatorId: string, @Body('chain') chain) {
    const aggregator = await loadAggregator(aggregatorId, chain)
    console.log(aggregator)

    if (Object.keys(aggregator).length == 0) {
      const msg = `Aggregator [${aggregatorId}] not found`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    }

    // TODO define aggregator type
    if (aggregator['active']) {
      const msg = `Aggregator [${aggregatorId}] is already active`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.BAD_REQUEST)
    }

    const adapter = aggregator['adapter']
    const feeds = extractFeeds(adapter, aggregator['id']) // FIXME define types

    // TODO Validate adapter

    // Launch recurrent data collection
    const job = await this.queue.add(aggregatorId, feeds, {
      repeat: {
        every: 2_000 // FIXME load env settings
      },
      removeOnComplete: true,
      removeOnFail: true
    })

    const msg = `Activated [${aggregatorId}]`
    this.logger.log(msg)
    return msg

    // TODO update aggregator to active = true
    // TODO log the command to separate table
  }

  @Get('stop/:aggregator')
  async stop(@Param('aggregator') id: string) {
    const delayed = await this.queue.getJobs(['delayed'])
    const filtered = delayed.filter((job) => job.name == id)

    if (filtered.length == 1) {
      const job = filtered[0]
      job.remove()
      // TODO update aggregator to active = false
      const msg = `Deactivated [${id}]`
      this.logger.log(msg)
      return msg
    } else if (filtered.length == 0) {
      const msg = `Job [${id}] does not exist`
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.NOT_FOUND)
    } else {
      const msg = 'Found more than one job satisfying your criteria'
      this.logger.error(msg)
      throw new HttpException(msg, HttpStatus.INTERNAL_SERVER_ERROR)
    }
  }
}
