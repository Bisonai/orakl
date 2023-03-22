import { Type } from 'cmd-ts'
import { existsSync } from 'node:fs'
import { CliError, CliErrorCode } from './errors'
import { isValidUrl, loadFile, loadJsonFromUrl } from './utils'

export const ReadFile: Type<string, string> = {
  async from(filePath) {
    if (await isValidUrl(filePath)) {
      // load from Url
      return await loadJsonFromUrl(filePath)
    } else {
      // load from Path
      if (!existsSync(filePath)) {
        throw new CliError(CliErrorCode.FileNotFound)
      }
      return JSON.parse((await loadFile(filePath)).toString())
    }
  }
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
