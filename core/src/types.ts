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
  REQUEST_RESPONSE: string[]
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
  id: string
  active: boolean
  name: string
  job_type: string
  feeds: IFeed[]
}

interface IProperty {
  active: boolean
  value: number
}

export interface IAggregator {
  id: string
  active: boolean
  name: string
  aggregatorAddress: string
  fixedHeartbeatRate: IProperty
  randomHeartbeatRate: IProperty
  threshold: number
  absoluteThreshold: number
  adapterId: string
}

export interface IRequestOperation {
  function: string
  args: string
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

export interface IOracleRoundState {
  _eligibleToSubmit: boolean
  _roundId: number
  _latestSubmission: BigNumber
  _startedAt: BigNumber
  _timeout: BigNumber
  _availableFunds: BigNumber
  _oracleCount: number
  _paymentAmount: BigNumber
}

// Events

export interface IDataRequested {
  requestId: BigNumber
  jobId: string
  accId: BigNumber
  callbackGasLimit: number
  sender: string
  isDirectPayment: boolean
  data: string
}

export interface IRandomWordsRequested {
  keyHash: string
  requestId: BigNumber
  preSeed: number
  accId: BigNumber
  callbackGasLimit: number
  numWords: number
  sender: string
  isDirectPayment: boolean
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

export interface IRequestResponseListenerWorker {
  callbackAddress: string
  blockNum: number
  requestId: string
  jobId: string
  accId: string
  callbackGasLimit: number
  sender: string
  isDirectPayment: boolean
  data: string
}

export interface IVrfListenerWorker {
  callbackAddress: string
  blockNum: string
  blockHash: string
  requestId: string
  seed: string
  accId: string
  callbackGasLimit: number
  numWords: number
  sender: string
  isDirectPayment: boolean
}

export interface IAggregatorListenerWorker {
  address: string
  roundId: BigNumber
  startedBy: string
  startedAt: BigNumber
}

// Worker -> Worker

export interface IAggregatorHeartbeatWorker {
  id: string
  address: string
  name: string
  active: boolean
  report?: boolean
  fixedHeartbeatRate: IProperty
  randomHeartbeatRate: IProperty
  threshold: number
  absoluteThreshold: number
  adapterId: string
  adapter: IFeed[]
}

// Worker -> Reporter

export interface IRequestResponseWorkerReporter {
  callbackAddress: string
  blockNum: number
  requestId: string
  jobId: string
  accId: string
  callbackGasLimit: number
  sender: string
  isDirectPayment: boolean
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
  accId: string
  callbackGasLimit: number
  numWords: number
  sender: string
  isDirectPayment: boolean
  pk: [string, string]
  proof: [string, string, string, string]
  preSeed: string
  uPoint: [string, string]
  vComponents: [string, string, string, string]
}

export interface IAggregatorWorkerReporter {
  report: boolean | undefined
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

export type RequestCommitmentVRF = [
  string /* blockNum */,
  string /* accId */,
  number /* callbackGasLimit */,
  number /* numWords */,
  string /* sender */
]

export type RequestCommitmentRequestResponse = [
  number /* blockNum */,
  string /* accId */,
  number /* callbackGasLimit */,
  string /* sender */
]

export interface IListenerBlock {
  startBlock: number
  filePath: string
}

export interface IListenerConfig {
  address: string
  eventName: string
}

export interface IVrfConfig {
  sk: string
  pk: string
  pk_x: string
  pk_y: string
}
