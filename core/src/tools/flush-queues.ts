import { Queue } from 'bullmq'
import { ALL_QUEUES, BULLMQ_CONNECTION } from '../settings'

async function main() {
  for (const q of ALL_QUEUES) {
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
