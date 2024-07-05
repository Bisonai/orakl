import { Type } from 'cmd-ts'
import { existsSync } from 'node:fs'
import { CliError, CliErrorCode } from './errors.js'
import { isValidUrl, loadFile, loadJsonFromUrl } from './utils.js'

export async function readFileFromSource(source: string) {
  if (await isValidUrl(source)) {
    return await loadJsonFromUrl(source)
  } else {
    if (!existsSync(source)) {
      throw new CliError(CliErrorCode.FileNotFound)
    }
    return JSON.parse((await loadFile(source)).toString())
  }
}

// ReadFile function is to load json file from
// url-link, or from local file directory
export const ReadFile: Type<string, string> = {
  async from(source) {
    return await readFileFromSource(source)
  },
}

export interface ITransaction {
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
  adapterHash?: string
  name: string
  decimals: number
  feeds: IFeed[]
}

export interface IAggregator {
  aggregatorHash?: string
  active: boolean
  name: string
  address: string
  heartbeat: number
  threshold: number
  absoluteThreshold: number
  adapterHash: string
}

export interface IDatafeedBulkInsertElement {
  adapterSource: string
  aggregatorSource: string
  reporter: {
    walletAddress: string
    walletPrivateKey: string
  }
}

export interface IDatafeedBulk {
  chain?: string
  service?: string
  organization?: string
  functionName?: string
  eventName?: string
  fetcherHost?: string
  workerHost?: string
  listenerHost?: string
  reporterHost?: string
  fetcherPort?: string
  workerPort?: string
  listenerPort?: string
  reporterPort?: string
  bulk: IDatafeedBulkInsertElement[]
}
