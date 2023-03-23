import { Logger } from 'pino'
import type { RedisClientType } from 'redis'

export interface IReporters {
  [index: string]: (redisClient: RedisClientType, _logger: Logger) => Promise<void>
}
