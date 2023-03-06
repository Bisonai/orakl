import { Controller, Get, Param } from '@nestjs/common'
import { JobDto } from './dto/job.dto'
import { InjectQueue } from '@nestjs/bullmq'
import { Queue } from 'bullmq'
import { Logger } from '@nestjs/common'

@Controller({
  version: '1'
})
export class JobController {
  private readonly logger = new Logger(JobController.name)
  constructor(@InjectQueue('orakl-fetcher-queue') private queue: Queue) {}

  @Get('start/:aggregator')
  async start(@Param('aggregator') id: string) {
    console.log('added job')
    // TODO check aggregator
    const job = await this.queue.add(
      id,
      { foo: 'bar' },
      {
        repeat: {
          every: 1_000
        },
        removeOnComplete: true,
        removeOnFail: true
      }
    )
  }

  @Get('stop/:aggregator')
  async stop(@Param('aggregator') id: string) {
    const delayed = await this.queue.getJobs(['delayed'])
    const filtered = delayed.filter((job) => job.name == id)

    if (filtered.length == 1) {
      const job = filtered[0]
      job.remove()
      this.logger.log(`Job ${id} removed`)
    } else if (filtered.length == 0) {
      this.logger.error(`job ${id} does not exist`)
    } else {
    }
  }
}
