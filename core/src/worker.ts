import { Worker } from 'bullmq'
import { buildBullMqConnection, buildQueueName } from './utils'

const worker = new Worker(
  buildQueueName(),
  async (job) => {
    console.log(job.data)
  },
  buildBullMqConnection()
)
