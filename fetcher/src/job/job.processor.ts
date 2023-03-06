import { Processor, WorkerHost, OnWorkerEvent } from '@nestjs/bullmq'
import { Job } from 'bullmq'

@Processor('orakl-fetcher-queue')
export class JobProcessor extends WorkerHost {
  async process(job: Job<any, any, string>): Promise<any> {
    console.log('working')
  }

  @OnWorkerEvent('completed')
  onCompleted() {
    console.log('completed')
  }
}
