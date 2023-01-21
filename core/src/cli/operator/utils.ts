import {
  optional,
  boolean as cmdboolean,
  number as cmdnumber,
  string as cmdstring,
  flag,
  option
} from 'cmd-ts'
import sqlite from 'sqlite3'
import { open } from 'sqlite'
import { CliError, CliErrorCode } from './error'
import { ChainId, ServiceId, DbCmdOutput } from './types'
import { SETTINGS_DB_FILE } from '../../settings'

export async function openDb({ dbFile, migrate }: { dbFile?: string; migrate?: boolean }) {
  const db = await open({
    filename: dbFile || SETTINGS_DB_FILE,
    driver: sqlite.Database
  })

  if (migrate) {
    await db.migrate({ force: true })
  }

  return db
}

export async function chainToId(db, chain: string) {
  const query = `SELECT id from Chain WHERE name='${chain}';`
  const result: ChainId = await db.get(query)
  if (!result) {
    throw new CliError(CliErrorCode.NonExistentChain)
  }
  return result.id
}

export async function serviceToId(db, service: string) {
  const query = `SELECT id from Service WHERE name='${service}';`
  const result: ServiceId = await db.get(query)
  if (!result) {
    throw new CliError(CliErrorCode.NonExistentService)
  }
  return result.id
}

export const chainOptionalOption = option({
  type: optional(cmdstring),
  long: 'chain'
})

export const serviceOptionalOption = option({
  type: optional(cmdstring),
  long: 'service'
})

export const idOption = option({
  type: cmdnumber,
  long: 'id'
})

export const dryrunOption = flag({
  type: cmdboolean,
  long: 'dry-run'
})

export function formatResultInsert(output: DbCmdOutput): string {
  const row = output.changes == 1 ? 'row' : 'rows'
  return `Inserted ${output.changes} ${row}. The row id is ${output.lastID}.`
}

export function formatResultRemove(output: DbCmdOutput): string {
  const row = output.changes == 1 ? 'row' : 'rows'
  return `Removed ${output.changes} ${row}.`
}
