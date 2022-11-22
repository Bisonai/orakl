import { Worker } from 'bullmq'
import { buildBullMqConnection } from './utils'

const worker = new Worker(
  'foo',
  async (job) => {
    console.log(job.data)
  },
  buildBullMqConnection()
)
