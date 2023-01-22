import { Type } from 'cmd-ts'
import { existsSync } from 'node:fs'
import { CliError, CliErrorCode } from './error'
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
