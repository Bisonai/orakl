import { open as openFile, readFile } from 'node:fs/promises'
import * as fs from 'node:fs'
import path from 'node:path'
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
import { ethers } from 'ethers'
import { IAdapter, IAggregator } from './types'
import { CliError, CliErrorCode } from './errors'
import { ChainId, ServiceId, DbCmdOutput } from './cli-types'

function mkdir(dir: string) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

export async function openDb({
  dbFile,
  migrate,
  migrationsPath
}: {
  dbFile: string
  migrate?: boolean
  migrationsPath?: string
}) {
  mkdir(path.dirname(dbFile))

  const db = await open({
    filename: dbFile,
    driver: sqlite.Database
  })

  if (migrate) {
    await db.migrate({ force: true, migrationsPath })
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

export async function loadFile(filePath: string) {
  const f = await openFile(filePath)
  return readFile(f)
}

export async function computeDataHash({
  data,
  verify
}: {
  data: IAdapter | IAggregator
  verify?: boolean
}): Promise<IAdapter | IAggregator> {
  const input = JSON.parse(JSON.stringify(data))

  // Don't use `id` and `active` in hash computation
  delete input.id
  delete input.active
  delete input.address

  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

  if (verify && data.id != hash) {
    console.info(input)
    throw Error(`Hashes do not match!\nExpected ${hash}, received ${data.id}.`)
  } else {
    data.id = hash
    console.info(data)
    return data
  }
}

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}
