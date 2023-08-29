import { open as openFile, readFile } from 'node:fs/promises'
import axios from 'axios'
import { optional, number as cmdnumber, string as cmdstring, option } from 'cmd-ts'
import { ORAKL_NETWORK_API_URL, ORAKL_NETWORK_DELEGATOR_URL } from './settings'

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

export async function loadFile(source: string) {
  const f = await openFile(source)
  return readFile(f)
}

export async function loadJsonFromUrl(url: string) {
  return await (await fetch(url, { method: 'GET' })).json()
}

export async function isValidUrl(url: string) {
  try {
    new URL(url)
    return true
  } catch (e) {
    return false
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

export async function isOraklFetcherHealthy(url: string) {
  try {
    return 200 === (await axios.get(url))?.status
  } catch (e) {
    console.error(`Orakl Network Fetcher [${url}] is down`)
    return false
  }
}

export async function isOraklDelegatorHealthy() {
  try {
    return 200 === (await axios.get(ORAKL_NETWORK_DELEGATOR_URL))?.status
  } catch (e) {
    console.error(`Orakl Network Delegator [${ORAKL_NETWORK_DELEGATOR_URL}] is down`)
    return false
  }
}

export async function isServiceHealthy(url: string) {
  const healthEndpoint = buildUrl(url, 'health')
  try {
    return 200 === (await axios.get(healthEndpoint))?.status
  } catch (e) {
    console.log(e)
    console.error(`${healthEndpoint} is down`)
    return false
  }
}
