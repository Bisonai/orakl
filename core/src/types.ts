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
  method: string
  headers?: IHeader[]
  reducers?: IReducer[]
}

export interface IAdapter {
  active: boolean
  name: string
  job_type: string
  adapter_id: string
  feeds: IFeed[]
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
