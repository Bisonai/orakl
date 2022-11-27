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

// FIXME create a settings file instead
export function buildQueueName() {
  return 'worker-request-queue'
}

// FIXME create a settings file instead
// TODO move adapter out of src directory
export function buildAdapterRootDir() {
  return './src/adapter/'
}

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}
