import { Module } from '@nestjs/common'
import { BullModule } from '@nestjs/bullmq'
import { JobProcessor } from './job.processor'
import { JobController } from './job.controller'

@Module({
  imports: [
    BullModule.registerQueue({
      name: 'orakl-fetcher-queue',
      connection: {
        host: process.env.REDIS_HOST || 'localhost',
        port: Number(process.env.REDIS_PORT) || 6379
      }
    })
  ],
  controllers: [JobController],
  providers: [JobProcessor]
})
export class JobModule {}
