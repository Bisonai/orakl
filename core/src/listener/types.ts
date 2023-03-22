import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { IListenerConfig } from '../types'

export interface IListeners {
  [index: string]: (config: IListenerConfig[], redisClient: RedisClientType, logger: Logger) => void
}
