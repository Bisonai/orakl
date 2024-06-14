import { Queue } from 'bullmq'
import { BigNumber } from 'ethers'

interface IOraklRequest {
  id: string
  callbackAddr: string
  callbackFunc: string
  nonce: number
  buf: Buffer
}

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

export interface ILatestRoundData {
  roundId: BigNumber
  answer: BigNumber
  startedAt: BigNumber
  updatedAt: BigNumber
  answeredInRound: BigNumber
}

export interface IOracleRoundState {
  eligibleToSubmit: boolean
  roundId: number
  latestSubmission: BigNumber
  startedAt: BigNumber
  timeout: BigNumber
  availableFunds: BigNumber
  oracleCount: number
  paymentAmount: BigNumber
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
  numSubmission: number
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

export interface IRandomWordsFulfilled {
  requestId: BigNumber
  l2RequestId: BigNumber
  randomWords: BigNumber[]
  sender: string
  callbackGasLimit: number
}

export interface INewRound {
  roundId: BigNumber
  startedBy: string
  startedAt: BigNumber
}

export interface IAnswerUpdated {
  current: BigNumber
  roundId: BigNumber
  updatedAt: BigNumber
}

export interface IL2DataRequested {
  requestId: BigNumber
  jobId: string
  accId: BigNumber
  callbackGasLimit: number
  sender: string
  numSubmission: number
  req: IOraklRequest
}

export interface IL2DataRequestFulfilled {
  requestId: BigNumber
  l2RequestId: BigNumber
  sender: string
  callbackGasLimit: number
  jobId: string
  responseUint128: BigNumber
  responseInt256: BigNumber
  responseBool: boolean
  responseString: string
  responseBytes32: string
  responseBytes: string
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
  numSubmission: number
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

export interface IL2EndpointListenerWorker {
  keyHash: string
  callbackAddress: string
  blockNum: string
  blockHash: string
  requestId: string
  seed: string
  accId: string
  callbackGasLimit: number
  numWords: number
  sender: string
}

export interface IDataFeedListenerWorker {
  oracleAddress: string
  roundId: number
  workerSource: string
}

export interface IDataFeedListenerWorkerL2 {
  oracleAddress: string
  roundId: number
  answer: number
  workerSource: string
}

export interface IL2VrfFulfillListenerWorker {
  callbackAddress: string
  callbackGasLimit: number
  blockNum: string
  blockHash: string
  requestId: string
  l2RequestId: string
  sender: string
  randomWords: string[]
}

export interface IL2RequestResponseListenerWorker {
  callbackAddress: string
  blockNum: string
  requestId: string
  jobId: string
  accId: string
  callbackGasLimit: number
  sender: string
  numSubmission: number
  req: IOraklRequest
}

export interface IL2RequestResponseFulfillListenerWorker {
  callbackAddress: string
  blockNum: number
  requestId: string
  jobId: string
  callbackGasLimit: number
  sender: string
  l2RequestId: string
  response: string | boolean
}

// Worker -> Worker

export interface IAggregatorHeartbeatWorker {
  oracleAddress: string
}

export interface IAggregatorSubmitHeartbeatWorker {
  oracleAddress: string
  delay: number
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
  number /* numSubmission */,
  number /* callbackGasLimit */,
  string /* sender */,
  boolean /* isDirectPayment */,
  string /* jobId */
]

export interface IVrfConfig {
  sk: string
  pk: string
  pkX: string
  pkY: string
  keyHash: string
}

// Listener
export interface IListenerRawConfig {
  address: string
  eventName: string
  service: string
  chain?: string
}

export interface IListenerConfig {
  id: string
  address: string
  eventName: string
  chain: string
}

export interface IListenerGroupConfig {
  [key: string]: IListenerConfig[]
}

// Reporter
export interface IReporterConfig {
  id: string
  address: string
  privateKey: string
  oracleAddress: string
  chain: string
  service: string
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
  id: string
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
  timestamp: string
  value: bigint
  aggregatorId: string
}

export interface IAggregateById {
  timestamp: string
  value: bigint
}

export interface ITransactionParameters {
  payload: string
  gasLimit: number | string
  to: string
}

export interface ITransactionParametersWithNonce extends ITransactionParameters {
  nonce: number
}

export interface IVrfTransactionParameters {
  blockNum: string
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

export interface IL2VrfRequestTransactionParameters {
  keyHash: string
  blockNum: string
  seed: string
  accId: string
  callbackGasLimit: number
  numWords: number
  sender: string
  l2RequestId: string
}

export interface IL2VrfFulfillTransactionParameters {
  requestId: string
  callbackGasLimit: number
  randomWords: string[]
}

export interface IL2RequestResponseRequestTransactionParameters {
  blockNum: string
  accId: string
  callbackGasLimit: number
  numSubmission: number
  sender: string
  l2RequestId: string
  req: IOraklRequest
}

export interface IL2RequestResponseFulfillTransactionParameters {
  requestId: string
  jobId: string
  callbackGasLimit: number
  response: string | boolean
}

export interface IRequestResponseTransactionParameters {
  blockNum: number
  accId: string
  jobId: string
  requestId: string
  numSubmission: number
  callbackGasLimit: number
  sender: string
  isDirectPayment: boolean
  response: any // eslint-disable-line @typescript-eslint/no-explicit-any
}

export interface IDataFeedTransactionParameters {
  roundId: number
  submission: bigint
}

export interface MockQueue {
  add: any // eslint-disable-line @typescript-eslint/no-explicit-any
  process: any // eslint-disable-line @typescript-eslint/no-explicit-any
  on: any // eslint-disable-line @typescript-eslint/no-explicit-any
}

export type QueueType = Queue | MockQueue

// Delegated Fee
export interface ITransactionData {
  from: string
  to: string
  input: string
  gas: string
  value: string
  chainId: string
  gasPrice: string
  nonce: string
  v: string
  r: string
  s: string
  rawTx: string
}

export interface IErrorMsgData {
  requestId: string
  timestamp: Date | string
  code: string
  name: string
  stack: string
}

export interface IL2AggregatorPair {
  id: bigint
  l1AggregatorAddress: string
  l2AggregatorAddress: string
  chainId: bigint
  active: boolean
}
