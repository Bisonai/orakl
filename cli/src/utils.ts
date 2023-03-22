import { open as openFile, readFile } from 'node:fs/promises'
import * as fs from 'node:fs'
import path from 'node:path'
import os from 'node:os'
import axios from 'axios'
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
import { CliError, CliErrorCode } from './errors'
import { ChainId, ServiceId, DbCmdOutput } from './cli-types'
import { ORAKL_NETWORK_API_URL, ORAKL_NETWORK_FETCHER_URL } from './settings'

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

export async function loadJsonFromUrl(url: string) {
  const res = await (await fetch(url, { method: 'Get' })).json()
  return res
}

export async function isValidUrl(url: string) {
  try {
    return Boolean(new URL(url))
  } catch (e) {
    return false
  }
}

export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

export async function isOraklNetworkApiHealthy() {
  const ORAKL_NETWORK_API_HEALTH_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'health')
  try {
    return 'OK' === (await axios.get(ORAKL_NETWORK_API_HEALTH_ENDPOINT))?.data
  } catch (e) {
    console.error(`Orakl Network API [${ORAKL_NETWORK_API_URL}] is down`)
    return false
  }
}

export async function isOraklFetcherHealthy() {
  const ORAKL_NETWORK_FETCHER_ENDPOINT = buildUrl(ORAKL_NETWORK_FETCHER_URL, 'health')
  try {
    return 'OK' === (await axios.get(ORAKL_NETWORK_FETCHER_ENDPOINT))?.data
  } catch (e) {
    console.error(`Orakl Network Fetcher [${ORAKL_NETWORK_FETCHER_URL}] is down`)
    return false
  }
}

export function mkTmpFile({ fileName }: { fileName: string }): string {
  const appPrefix = 'orakl'
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), appPrefix))
  const tmpFilePath = path.join(tmpDir, fileName)
  return tmpFilePath
}
