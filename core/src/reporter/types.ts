import { NonceManager } from '@ethersproject/experimental'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { CaverWallet } from './utils'

export interface IReporters {
  [index: string]: (redisClient: RedisClientType, _logger: Logger) => Promise<void>
}

export type Wallet = CaverWallet | NonceManager
