export interface IRawData {
  id: bigint
  value: number
}

export interface IData {
  aggregatorId: string
  feedId: bigint
  timestamp: string
  value: number
}

interface IHeader {
  'Content-Type': string
}

interface IReducer {
  function: string
  args: string[]
}

interface IDefinition {
  url: string
  method: string
  headers: IHeader[]
  reducers: IReducer[]
}

interface IFeed {
  id: bigint
  name: string
  definition: IDefinition
  adapterId: bigint
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
  adapter: IAdapter
  fetcherType: bigint
}

export interface IFetchedData {
  id: string
  value: number
}
export interface IAggregate {
  id: bigint
  timestamp: string
  value: bigint
  aggregatorId: bigint
}
export interface IDeviationData {
  timestamp: string
  submission: number
  oracleAddress: string
}
