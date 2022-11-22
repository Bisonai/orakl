import * as dotenv from 'dotenv'

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
