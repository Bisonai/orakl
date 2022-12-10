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

export interface IVrfRequest {
  alpha: string
}

export interface IVrfResponse {
  pk: [number, number]
  proof: [number, number, number, number]
  uPoint: [number, number]
  vComponents: [number, number, number, number]
}
