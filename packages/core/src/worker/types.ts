import { Logger } from 'pino'
import type { RedisClientType } from 'redis'

export interface IWorkers {
  [index: string]: (redisClient: RedisClientType, _logger: Logger) => Promise<void>
}

export interface IAggregatorConfig {
  id: string
  aggregatorHash: string
  name: string
  address: string
  heartbeat: number
  threshold: number
  absoluteThreshold: number
  chain: string
  timestamp: number
}
export interface IDeviationData {
  timestamp: string
  submission: bigint
  oracleAddress: string
}
