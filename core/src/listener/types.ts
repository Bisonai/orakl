import { ethers } from 'ethers'
import { Logger } from 'pino'
import type { RedisClientType } from 'redis'
import {
  IDataFeedListenerWorker,
  IDataFeedListenerWorkerL2,
  IL2EndpointListenerWorker,
  IL2RequestResponseFulfillListenerWorker,
  IL2RequestResponseListenerWorker,
  IL2VrfFulfillListenerWorker,
  IListenerConfig,
  IRequestResponseListenerWorker,
  IVrfListenerWorker,
} from '../types'

export interface IListeners {
  [index: string]: (config: IListenerConfig[], redisClient: RedisClientType, logger: Logger) => void
}

/**
 * Listener can be launched in one of three different initialization
 * approaches: [latest], [clear], or by specificing a [block number]
 * from which the listener starts the event tracking.
 *
 * [latest] initialization can be used when launching a new listener,
 * or when restarting listener without any special requirements. For
 * the initial launch, listener will start tracking event from the
 * latest block. For the subsequent restarts, the latest observed
 * block will be used as a starting point to continue event tracking.
 *
 * [clear] initialization removes metadata about previously observed
 * blocks, therefore every new launch of listener will start tracking
 * events from the current latest block at that time.
 *
 * [block number] initialization can be used to specify from which
 * block we want the listener to start event tracking.
 */
export type ListenerInitType = 'latest' | 'clear' | number

interface IJobQueueSettings {
  removeOnComplete?: number | boolean
  removeOnFail?: number | boolean
  attempts?: number
  backoff?: number
}

export type ProcessEventOutputType = {
  jobData:
    | IRequestResponseListenerWorker
    | IDataFeedListenerWorker
    | IVrfListenerWorker
    | IDataFeedListenerWorkerL2
    | IL2VrfFulfillListenerWorker
    | IL2EndpointListenerWorker
    | IL2RequestResponseListenerWorker
    | IL2RequestResponseFulfillListenerWorker
    | null
  jobId: string
  jobName: string
  jobQueueSettings?: IJobQueueSettings
}

export interface ILatestListenerJob {
  contractAddress: string
}

export interface IHistoryListenerJob {
  contractAddress: string
  blockNumber: number
}

export interface IProcessEventListenerJob {
  contractAddress: string
  events: ethers.Event[]
  blockNumber: number
}

export interface IContracts {
  [key: string]: ethers.Contract
}

export interface IBlock {
  service: string
  blockNumber: number
}
