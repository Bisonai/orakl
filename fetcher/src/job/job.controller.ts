import { Controller, Get, Post, Body, Patch, Param, Delete } from '@nestjs/common'
import { JobDto } from './dto/job.dto'
import { InjectQueue } from '@nestjs/bullmq'
import { Queue } from 'bullmq'

@Controller({
  version: '1'
})
export class JobController {
  constructor(@InjectQueue('orakl-fetcher-queue') private queue: Queue) {}

  @Get('start')
  async start() {
    console.log('added job')
    const job = await this.queue.add(
      'job-name',
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

  @Get('stop')
  async stop() {
    // TODO filter among all job
    const allDelayed = await this.queue.getJobs(['delayed'])
    const job = allDelayed[0]
    job.remove()
  }
}
