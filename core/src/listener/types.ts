import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import { IListenerConfig } from '../types'

export interface IListeners {
  [index: string]: (config: IListenerConfig[], redisClient: RedisClientType, logger: Logger) => void
}

export type ListenerInitType = 'latest' | 'observed' | number

export interface ILatestListenerJob {
  contractAddress: string
}

export interface IHistoryListenerJob {
  contractAddress: string
  blockNumber: number
}

export interface IProcessEventListenerJob {
  contractAddress: string
  event: { topics: Array<string>; data: string }
}

export interface IContracts {
  [key: string]: ethers.Contract
}
