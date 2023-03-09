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

// TODO deprecate
export interface IAdapter {
  id?: string
  active?: boolean
  name?: string
  jobType?: string
  decimals: number
  feeds: IFeed[]
}

interface IProperty {
  active: boolean
  value: number
}

// TODO deprecate
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

export interface IAggregatorWorker {
  aggregatorAddress: string
  roundId: number
  workerSource: string
}

// Worker -> Worker

export interface IAggregatorHeartbeatWorker {
  aggregatorAddress: string
}

export interface IAggregatorJob {
  id: string
  address: string
  name: string
  active: boolean
  report: boolean
  fixedHeartbeatRate: IProperty
  randomHeartbeatRate: IProperty
  threshold: number
  absoluteThreshold: number
  adapterId: string
  adapter: IFeed[]
  aggregatorAddress: string
  decimals: number
}

export interface IAggregatorMetadata {
  id: string
  address: string
  decimals: number
  threshold: number
  absoluteThreshold: number
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

// Data Feed
export interface IFeedDefinition {
  url: string
  method: string
  headers: IHeader[]
  reducers: IReducer[]
}

export interface IFeedNew {
  id: bigint
  adapterId: bigint
  name: string
  definition: IFeedDefinition
}

export interface IAdapterNew {
  id: bigint
  adapterHash: string
  name: string
  decimals: number
  feeds: IFeedNew[]
}

export interface IAggregatorNew {
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
  adapter: IAdapterNew
}

export interface IAggregate {
  id: bigint
  timestamp: string | Date
  value: bigint
  aggregatorId: bigint
}
