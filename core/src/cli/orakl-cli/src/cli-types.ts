import { Type } from 'cmd-ts'
import { existsSync } from 'node:fs'
import { CliError, CliErrorCode } from './errors'
import { loadFile } from './utils'

export interface ChainId {
  id: number
}

export interface ServiceId {
  id: number
}

export interface DbCmdOutput {
  lastID: number
  changes: number
}

export const ReadFile: Type<string, string> = {
  async from(filePath) {
    if (!existsSync(filePath)) {
      throw new CliError(CliErrorCode.FileNotFound)
    }

    return JSON.parse((await loadFile(filePath)).toString())
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
