import { Queue } from 'bullmq'
import { FETCHER_QUEUE_NAME } from '../settings'

export const BULLMQ_CONNECTION = {
  connection: {
    host: 'localhost',
    port: 6379,
  },
}

async function main() {
  const queues = [FETCHER_QUEUE_NAME]

  for (const q of queues) {
    process.stdout.write(`Flushing ${q}: `)
    const queue = new Queue(q, BULLMQ_CONNECTION)
    // await queue.drain((delayed = true))
    await queue.obliterate({ force: true })
    process.stdout.write('DONE\n')
  }

  process.exit(0)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
