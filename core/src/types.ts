import { BigNumber } from 'ethers'

export interface RequestEventData {
  specId: string
  requester: string
  payment: BigNumber
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

export interface IRoundData {
  roundId: BigNumber
  answer: BigNumber
  startedAt: BigNumber
  updatedAt: BigNumber
  answeredInRound: BigNumber
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

export interface IAggregatorWorker {
  aggregatorAddress: string
  roundId: number
  workerSource: string
}

// Worker -> Worker

export interface IAggregatorHeartbeatWorker {
  aggregatorAddress: string
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
  submission: bigint
  workerSource: string
  delay: number
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

export interface IListenerRawConfig {
  address: string
  eventName: string
  service: string
  chain?: string
}

export interface IListenerConfig {
  id?: string
  address: string
  eventName: string
  chain?: string
}

export interface IListenerGroupConfig {
  [key: string]: IListenerConfig[]
}

export interface IVrfConfig {
  sk: string
  pk: string
  pkX: string
  pkY: string
  keyHash: string
}

// Data Feed
interface IHeader {
  'Content-Type': string
}

interface IReducer {
  function: string
  args: string[]
}

export interface IFeedDefinition {
  url: string
  method: string
  headers: IHeader[]
  reducers: IReducer[]
}

export interface IFeed {
  id: bigint
  adapterId: bigint
  name: string
  definition: IFeedDefinition
}

export interface IAdapter {
  id: bigint
  adapterHash: string
  name: string
  decimals: number
  feeds: IFeed[]
}

export interface IAggregator {
  id: bigint
  aggregatorHash: string
  active: boolean
  name: string
  address: string
  heartbeat: number
  threshold: number
  absoluteThreshold: number
  adapterId: bigint
  chainId: bigint
  adapter?: IAdapter
}

export interface IAggregate {
  id: bigint
  timestamp: string | Date
  value: bigint
  aggregatorId: bigint
}
