import { Job } from 'bullmq'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { State } from './state'

export interface IReporters {
  [index: string]: (redisClient: RedisClientType, _logger: Logger) => Promise<void>
}

export type wrapperType = (job: Job) => Promise<void>

export type workerType = (state: State, logger: Logger) => wrapperType
