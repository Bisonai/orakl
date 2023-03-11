import { open as openFile, readFile } from 'node:fs/promises'
import * as fs from 'node:fs'
import path from 'node:path'
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
import { ethers } from 'ethers'
import { CliError, CliErrorCode } from './errors'
import { IAdapter, IAggregator, ChainId, ServiceId, DbCmdOutput } from './cli-types'
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

export async function computeAdapterHash({
  data,
  verify
}: {
  data: IAdapter
  verify?: boolean
}): Promise<IAdapter> {
  const input = JSON.parse(JSON.stringify(data))

  // Don't use following properties in computation of hash
  delete input.adapterHash

  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

  if (verify && data.adapterHash != hash) {
    throw new CliError(
      CliErrorCode.UnmatchingHash,
      `Hashes do not match!\nExpected ${hash}, received ${data.adapterHash}.`
    )
  } else {
    data.adapterHash = hash
    return data
  }
}

export async function computeAggregatorHash({
  data,
  verify
}: {
  data: IAggregator
  verify?: boolean
}): Promise<IAggregator> {
  const input = JSON.parse(JSON.stringify(data))

  // Don't use following properties in computation of hash
  delete input.aggregatorHash
  delete input.active
  delete input.address

  const hash = ethers.utils.keccak256(ethers.utils.toUtf8Bytes(JSON.stringify(input)))

  if (verify && data.aggregatorHash != hash) {
    throw new CliError(
      CliErrorCode.UnmatchingHash,
      `Hashes do not match!\nExpected ${hash}, received ${data.aggregatorHash}.`
    )
  } else {
    data.aggregatorHash = hash
    return data
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
