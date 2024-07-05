import { BullModule } from '@nestjs/bullmq'
import { Module } from '@nestjs/common'
import { DEVIATION_QUEUE_NAME, FETCHER_QUEUE_NAME } from 'src/settings'
import { JobController } from './job.controller'
import { JobProcessor } from './job.processor'

@Module({
  imports: [
    BullModule.registerQueue(
      {
        name: FETCHER_QUEUE_NAME,
        connection: {
          host: process.env.REDIS_HOST || 'localhost',
          port: Number(process.env.REDIS_PORT) || 6379,
        },
      },
      {
        name: DEVIATION_QUEUE_NAME,
        connection: {
          host: process.env.REDIS_HOST || 'localhost',
          port: Number(process.env.REDIS_PORT) || 6379,
        },
      },
    ),
  ],
  controllers: [JobController],
  providers: [JobProcessor],
})
export class JobModule {}
