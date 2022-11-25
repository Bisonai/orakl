import * as dotenv from 'dotenv'
import * as Fs from 'node:fs/promises'

dotenv.config()

export function buildBullMqConnection() {
  // FIXME Move to separate factory file?
  return {
    connection: {
      host: process.env.REDIS_HOST || 'localhost',
      port: Number(process.env.REDIS_PORT || 6379)
    }
  }
}

export function buildQueueName() {
  return 'worker-request-queue'
}

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}
