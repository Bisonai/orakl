import { IncomingWebhook } from '@slack/webhook'
import Hook from 'console-hook'
import * as Fs from 'node:fs/promises'
import os from 'node:os'
import type { RedisClientType } from 'redis'
import { createClient } from 'redis'
import { OraklErrorCode } from './errors'
import { SLACK_WEBHOOK_URL } from './settings'
import { QueueType } from './types'

export async function loadJson(filepath) {
  const json = await Fs.readFile(filepath, 'utf8')
  return JSON.parse(json)
}

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
    const e = error[0]
    const webhook = new IncomingWebhook(SLACK_WEBHOOK_URL)
    const text = ` :fire: _An error has occurred at_ \`${os.hostname()}\`\n \`\`\`${JSON.stringify(
      e
    )} \`\`\`\n>*System information*\n>*memory*: ${os.freemem()}/${os.totalmem()}\n>*machine*: ${os.machine()}\n>*platform*: ${os.platform()}\n>*upTime*: ${os.uptime()}\n>*version*: ${os.version()}
   `

    try {
      if (e && e.message && errMsg === e.message) {
        const now = new Date().getTime()
        if (slackSentTime + 60_000 < now) {
          await webhook.send({ text })
          slackSentTime = now
          errMsg = e.message
        }
      } else {
        await webhook.send({ text })
        if (e && e.message) errMsg = e.message
        slackSentTime = new Date().getTime()
      }
    } catch (e) {
      console.log('utils:sendToSlack', e)
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

export async function isRoundIdValid(queue: QueueType, oracleAddress: string, roundId: number) {
  const jobs = await queue.getJobs()
  const jobsWithSameOracleAddress = jobs.filter((_job) => _job.id?.split('-')[1] === oracleAddress)

  if (!jobsWithSameOracleAddress) {
    return true
  }

  let maxRoundId = 0
  for (const _job of jobsWithSameOracleAddress) {
    const _roundId = Number(_job.id?.split('-')[0]) || 0
    maxRoundId = Math.max(maxRoundId, _roundId)
  }

  return maxRoundId < roundId
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

// axios errors are defined in official repo (https://github.com/axios/axios#error-types)
export const getOraklErrorCode = (e, defaultErrorCode) => {
  if (e.code == 'ERR_BAD_OPTION_VALUE') {
    return OraklErrorCode.AxiosBadOptionValue
  } else if (e.code == 'ERR_BAD_OPTION') {
    return OraklErrorCode.AxiosBadOption
  } else if (e.code == 'ECONNABORTED' || e.code == 'ETIMEDOUT') {
    return OraklErrorCode.AxiosTimeOut
  } else if (e.code == 'ERR_NETWORK') {
    return OraklErrorCode.AxiosNetworkError
  } else if (e.code == 'ERR_FR_TOO_MANY_REDIRECTS') {
    return OraklErrorCode.AxiosTooManyRedirects
  } else if (e.code == 'ERR_DEPRECATED') {
    return OraklErrorCode.AxiosDeprecated
  } else if (e.code == 'ERR_BAD_RESPONSE') {
    return OraklErrorCode.AxiosBadResponse
  } else if (e.code == 'ERR_BAD_REQUEST') {
    return OraklErrorCode.AxiosBadRequest
  } else if (e.code == 'ERR_CANCELED') {
    return OraklErrorCode.AxiosCanceledByUser
  } else if (e.code == 'ERR_NOT_SUPPORT') {
    return OraklErrorCode.AxiosNotSupported
  } else if (e.code == 'ERR_INVALID_URL') {
    return OraklErrorCode.AxiosInvalidUrl
  } else {
    return defaultErrorCode
  }
}
