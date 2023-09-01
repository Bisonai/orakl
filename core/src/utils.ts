import * as Fs from 'node:fs/promises'
import os from 'node:os'
import { createClient } from 'redis'
import type { RedisClientType } from 'redis'
import Hook from 'console-hook'
import { MESSENGER_ENDPOINT, NODE_ENV } from './settings'
import { IMessageTransfer } from './types'
import axios from 'axios'

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

async function sendToSlack(method, error) {
  if (MESSENGER_ENDPOINT) {
    const e = error[1]
    const text = ` :fire: _An error has occurred at_ \`${os.hostname()}\`\n \`\`\`${JSON.stringify(
      e
    )} \`\`\`\n>*System information*\n>*memory*: ${os.freemem()}/${os.totalmem()}\n>*machine*: ${os.machine()}\n>*platform*: ${os.platform()}\n>*upTime*: ${os.uptime()}\n>*version*: ${os.version()}
   `

    const methodMap = {
      error: 'error-report',
      warn: 'warning-report'
    }
    const networkMap = {
      cypress: 'prod',
      baobab: 'prod',
      CYPRESS: 'prod',
      BAOBAB: 'prod'
    }
    const payload: IMessageTransfer = {
      company: 'bisonai',
      operator: 'bisonai',
      messageKind: methodMap[method],
      system: 'k8s',
      nodeEnv: networkMap[NODE_ENV || ''] || 'local',
      network: NODE_ENV || '',
      serviceName: 'monitor-api',
      message: text
    }

    try {
      if (errMsg === e.message) {
        const currentDate = new Date()
        const oneMinuteAgo = new Date(currentDate.getTime() - 60000)
        if (slackSentTime < oneMinuteAgo.getTime()) {
          axios.post(MESSENGER_ENDPOINT, payload, { timeout: 5000 })
          errMsg = error[1].message
          slackSentTime = new Date().getTime()
        }
      } else {
        axios.post(MESSENGER_ENDPOINT, payload, { timeout: 5000 })
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
      sendToSlack(method, args)
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
