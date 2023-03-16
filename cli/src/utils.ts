import { open as openFile, readFile } from 'node:fs/promises'
import axios from 'axios'
import { optional, number as cmdnumber, string as cmdstring, option } from 'cmd-ts'
import { ethers } from 'ethers'
import { CliError, CliErrorCode } from './errors'
import { IAdapter, IAggregator } from './cli-types'
import { ORAKL_NETWORK_API_URL, ORAKL_NETWORK_FETCHER_URL } from './settings'

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
  try {
    return 200 === (await axios.get(ORAKL_NETWORK_API_URL))?.status
  } catch (e) {
    console.error(`Orakl Network API [${ORAKL_NETWORK_API_URL}] is down`)
    return false
  }
}

export async function isOraklFetcherHealthy() {
  try {
    return 200 === (await axios.get(ORAKL_NETWORK_FETCHER_URL))?.status
  } catch (e) {
    console.error(`Orakl Network Fetcher [${ORAKL_NETWORK_FETCHER_URL}] is down`)
    return false
  }
}
