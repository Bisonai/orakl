import { Controller, Get, Param, HttpStatus, HttpException } from '@nestjs/common'
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
    const msg = `Added [${id}]`
    this.logger.log(msg)
    return msg
  }

  @Get('stop/:aggregator')
  async stop(@Param('aggregator') id: string) {
    const delayed = await this.queue.getJobs(['delayed'])
    const filtered = delayed.filter((job) => job.name == id)

    if (filtered.length == 1) {
      const job = filtered[0]
      job.remove()
      const msg = `Removed [${id}]`
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
