import { IncomingWebhook } from '@slack/webhook'
import axios from 'axios'
import Hook from 'console-hook'
import * as Fs from 'node:fs/promises'
import os from 'node:os'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { OraklError, OraklErrorCode } from './errors'
import { DELEGATOR_TIMEOUT, SLACK_WEBHOOK_URL } from './settings'

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}

// https://medium.com/javascript-scene/reduce-composing-software-fe22f0c39a1d
export const pipe =
  (...fns) =>
  (x) =>
    fns.reduce((v, f) => f(v), x)

export function remove0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s.substring(2)
  }
}

export function add0x(s) {
  if (s.substring(0, 2) == '0x') {
    return s
  } else {
    return '0x' + s
  }
}

export function pad32Bytes(data) {
  data = remove0x(data)
  let s = String(data)
  while (s.length < (64 || 2)) {
    s = '0' + s
  }
  return s
}

let slackSentTime = new Date().getTime()
let errMsg = null

async function sendToSlack(error) {
  if (SLACK_WEBHOOK_URL) {
    const e = error[1]
    const webhook = new IncomingWebhook(SLACK_WEBHOOK_URL)
    const text = ` :fire: _An error has occurred at_ \`${os.hostname()}\`\n \`\`\`${JSON.stringify(
      e
    )} \`\`\`\n>*System information*\n>*memory*: ${os.freemem()}/${os.totalmem()}\n>*machine*: ${os.machine()}\n>*platform*: ${os.platform()}\n>*upTime*: ${os.uptime()}\n>*version*: ${os.version()}
   `
    try {
      if (errMsg == e.message) {
        const currentDate = new Date()
        const oneMinuteAgo = new Date(currentDate.getTime() - 60000)
        if (slackSentTime < oneMinuteAgo.getTime()) {
          await webhook.send({ text })
          errMsg = error[1].message
          slackSentTime = new Date().getTime()
        }
      } else {
        await webhook.send({ text })
        errMsg = e.message
        slackSentTime = new Date().getTime()
      }
    } catch (e) {
      console.log('utils:sendToSlack', `${e}`)
    }
  }
}

export function hookConsoleError(logger) {
  const consoleHook = Hook(logger).attach((method, args) => {
    if (method == 'error') {
      sendToSlack(args)
    }
  })
  consoleHook.detach
}

export async function createRedisClient(host: string, port: number): Promise<RedisClientType> {
  const client: RedisClientType = createClient({
    // redis[s]://[[username][:password]@][host][:port][/db-number]
    url: `redis://${host}:${port}`
  })
  await client.connect()
  return client
}

export function buildSubmissionRoundJobId({
  oracleAddress,
  roundId,
  deploymentName
}: {
  oracleAddress: string
  roundId: number
  deploymentName: string
}) {
  return `${roundId}-${oracleAddress}-${deploymentName}`
}

export function buildHeartbeatJobId({
  oracleAddress,
  deploymentName
}: {
  oracleAddress: string
  deploymentName: string
}) {
  return `${oracleAddress}-${deploymentName}`
}

/*
 * Connect `host` and `path` to a single url string, and remove all
 * duplicates of `/` (= slash character) except the first occurrence.
 *
 * @param {string} host, presumably includes scheme string `http(s)://`
 * @param {string} endpoint path
 * @return {string} concatenated string composed of host and endpoint path
 */
export function buildUrl(host: string, path: string) {
  const url = [host, path].join('/')
  return url.replace(/([^:]\/)\/+/g, '$1')
}

const axiosWithTimeOut = axios.create({ timeout: Number(DELEGATOR_TIMEOUT) })

// axios errors defined in official repo (https://github.com/axios/axios#error-types)
const handleAxiosError = (e) => {
  if (e.code == 'ERR_BAD_OPTION_VALUE') {
    throw new OraklError(OraklErrorCode.AxiosBadOptionValue)
  } else if (e.code == 'ERR_BAD_OPTION') {
    throw new OraklError(OraklErrorCode.AxiosBadOption)
  } else if (e.code == 'ECONNABORTED' || e.code == 'ETIMEDOUT') {
    throw new OraklError(OraklErrorCode.AxiosTimeOut)
  } else if (e.code == 'ERR_NETWORK') {
    throw new OraklError(OraklErrorCode.AxiosNetworkError)
  } else if (e.code == 'ERR_FR_TOO_MANY_REDIRECTS') {
    throw new OraklError(OraklErrorCode.AxiosTooManyRedirects)
  } else if (e.code == 'ERR_DEPRECATED') {
    throw new OraklError(OraklErrorCode.AxiosDeprecated)
  } else if (e.code == 'ERR_BAD_RESPONSE') {
    throw new OraklError(OraklErrorCode.AxiosBadResponse)
  } else if (e.code == 'ERR_BAD_REQUEST') {
    throw new OraklError(OraklErrorCode.AxiosBadRequest)
  } else if (e.code == 'ERR_CANCELED') {
    throw new OraklError(OraklErrorCode.AxiosCanceledByUser)
  } else if (e.code == 'ERR_NOT_SUPPORT') {
    throw new OraklError(OraklErrorCode.AxiosNotSupported)
  } else if (e.code == 'ERR_INVALID_URL') {
    throw new OraklError(OraklErrorCode.AxiosInvalidUrl)
  } else {
    throw e
  }
}

export const axiosWrapper = {
  makeRequest: async (method, url, data = null) => {
    try {
      const response = await axiosWithTimeOut({
        method,
        url,
        data
      })

      console.log(`Request successful. Response:`, response.data)
      return response.data
    } catch (e) {
      handleAxiosError(e)
    }
  },
  get: (url, data) => {
    return axiosWrapper.makeRequest('get', url, data)
  },
  post: (url, data) => {
    return axiosWrapper.makeRequest('post', url, data)
  },
  delete: (url, data) => {
    return axiosWrapper.makeRequest('delete', url, data)
  },
  patch: (url, data) => {
    return axiosWrapper.makeRequest('patch', url, data)
  }
}
