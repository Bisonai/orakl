import axios from 'axios'
import { number as cmdnumber, option, optional, string as cmdstring } from 'cmd-ts'
import { open as openFile, readFile } from 'node:fs/promises'
import { ORAKL_NETWORK_API_URL, ORAKL_NETWORK_DELEGATOR_URL } from './settings.js'

export const chainOptionalOption = option({
  type: optional(cmdstring),
  long: 'chain',
})

export const serviceOptionalOption = option({
  type: optional(cmdstring),
  long: 'service',
})

export const fetcherTypeOptionalOption = option({
  type: optional(cmdnumber),
  long: 'fetcherType',
})

export const proxyOptionalOption = option({
  type: optional(cmdstring),
  long: 'location',
})

export const idOption = option({
  type: cmdnumber,
  long: 'id',
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

export function buildFetcherUrl(host: string, port: string, apiVersion: string) {
  return `${host}:${port}${apiVersion}`
}

export async function isOraklNetworkApiHealthy() {
  try {
    return 200 === (await axios.get(ORAKL_NETWORK_API_URL))?.status
  } catch (e) {
    console.error(`Orakl Network API [${ORAKL_NETWORK_API_URL}] is down`)
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
