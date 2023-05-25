import { Module } from '@nestjs/common'
import { BullModule } from '@nestjs/bullmq'
import { JobProcessor } from './job.processor'
import { JobController } from './job.controller'
import { DEVIATION_QUEUE_NAME, FETCHER_QUEUE_NAME } from 'src/settings'

@Module({
  imports: [
    BullModule.registerQueue(
      {
        name: FETCHER_QUEUE_NAME,
        connection: {
          host: process.env.REDIS_HOST || 'localhost',
          port: Number(process.env.REDIS_PORT) || 6379
        }
      },
      {
        name: DEVIATION_QUEUE_NAME,
        connection: {
          host: process.env.REDIS_HOST || 'localhost',
          port: Number(process.env.REDIS_PORT) || 6379
        }
      }
    )
  ],
  controllers: [JobController],
  providers: [JobProcessor]
})
export class JobModule {}
