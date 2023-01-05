import { BigNumber } from 'ethers'

export interface RequestEventData {
  specId: string
  requester: string
  payment: BigNumber
}

export interface DataFeedRequest {
  from: string
  specId: string
}

export interface IListeners {
  VRF: string[]
  AGGREGATORS: string[]
  ANY_API: string[]
}

export interface ILog {
  address: string
  blockHash: string
  blockNumber: string
  data: string
  logIndex: string
  removed: boolean
  topics: string[]
  transactionHash: string
  transactionIndex: string
}

interface IHeader {
  'Content-Type': string
}

interface IReducer {
  function: string
  args: string[]
}

interface IFeed {
  url: string
  headers?: IHeader[]
  method: string
  reducers?: IReducer[]
}

export interface IAdapter {
  active: boolean
  name: string
  job_type: string
  adapter_id: string
  feeds: IFeed[]
}

export interface IAggregator {
  active: boolean
  name: string
  aggregatorAddress: string
  fixedHeartbeatRate: number
  randomHeartbeatRate: number
  threshold: number
  absoluteThreshold: number
  adapterId: string
}

export interface IRequest {
  get: string
  path?: string[]
}

export interface IVrfResponse {
  pk: [string, string]
  proof: [string, string, string, string]
  uPoint: [string, string]
  vComponents: [string, string, string, string]
}

export interface ILatestRoundData {
  roundId: BigNumber
  answer: BigNumber
  startedAt: BigNumber
  updatedAt: BigNumber
  answeredInRound: BigNumber
}

// Events

export interface INewRequest {
  requestId: string
  jobId: string
  nonce: number
  callbackAddress: string
  callbackFunctionId: string
  _data: string
}

export interface IRandomWordsRequested {
  keyHash: string
  requestId: BigNumber
  preSeed: number
  subId: BigNumber
  minimumRequestConfirmations: number
  callbackGasLimit: number
  numWords: number
  sender: string
}

export interface INewRound {
  roundId: BigNumber
  startedBy: string
  startedAt: BigNumber
}

// Listener -> Worker

export interface IPredefinedFeedListenerWorker {
  requestId: string
  jobId: string
  nonce: string
  callbackAddress: string
  callbackFunctionId: string
  _data: string
}

export interface IAnyApiListenerWorker {
  oracleCallbackAddress: string
  requestId: string
  jobId: string
  nonce: string
  callbackAddress: string
  callbackFunctionId: string
  _data: string
}

export interface IVrfListenerWorker {
  callbackAddress: string
  blockNum: string
  blockHash: string
  requestId: string
  seed: string
  subId: string
  minimumRequestConfirmations: number
  callbackGasLimit: number
  numWords: number
  sender: string
}

export interface IAggregatorListenerWorker {
  mustReport: boolean
  callbackAddress: string
  roundId: BigNumber
  startedBy: string
  startedAt: BigNumber
}

// Worker -> Worker

export interface IAggregatorFixedHeartbeatWorker {
  name: string
  active: boolean
  aggregatorAddress: string
  fixedHeartbeatRate: number
  randomHeartbeatRate: number
  threshold: number
  absoluteThreshold: number
  adapterId: string
  aggregatorId: string
  adapter: IFeed[]
}

// Worker -> Reporter

export interface IAnyApiWorkerReporter {
  oracleCallbackAddress: string
  requestId: string
  jobId: string
  callbackAddress: string
  callbackFunctionId: string
  data: string | number
}

export interface IPredefinedFeedWorkerReporter {
  requestId: string
  jobId: string
  callbackAddress: string
  callbackFunctionId: string
  data: string | number
}

export interface IVrfWorkerReporter {
  callbackAddress: string
  blockNum: string
  requestId: string
  seed: string
  subId: string
  minimumRequestConfirmations: number
  callbackGasLimit: number
  numWords: number
  sender: string
  pk: [string, string]
  proof: [string, string, string, string]
  preSeed: string
  uPoint: [string, string]
  vComponents: [string, string, string, string]
}

export interface IAggregatorWorkerReporter {
  callbackAddress: string
  roundId: number
  submission: number
}

// VRF
export type Proof = [
  [string, string] /* pk */,
  [string, string, string, string] /* proof */,
  string /* preSeed */,
  [string, string] /* uPoint */,
  [string, string, string, string] /* vComponents */
]

export type RequestCommitment = [
  string /* blockNum */,
  string /* subId */,
  number /* callbackGasLimit */,
  number /* numWords */,
  string /* sender */
]

export interface IListenerBlock {
  startBlock: number
  filePath: string
}

export interface IListenerConfig {
  address: string
  eventName: string
  factoryName: string
}
