import * as Fs from 'node:fs/promises'
import * as fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { createClient } from 'redis'
import type { RedisClientType } from 'redis'
import { IncomingWebhook } from '@slack/webhook'
import Hook from 'console-hook'
import { SLACK_WEBHOOK_URL } from './settings'
import urlExist from 'url-exist'
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

export function mkdir(dir: string) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true })
  }
}

export async function readTextFile(filepath: string) {
  return await Fs.readFile(filepath, 'utf8')
}

export async function writeTextFile(filepath: string, content: string) {
  await Fs.writeFile(filepath, content)
}

export function mkTmpFile({ fileName }: { fileName: string }): string {
  const appPrefix = 'orakl'
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), appPrefix))
  const tmpFilePath = path.join(tmpDir, fileName)
  return tmpFilePath
}

let slackSentTime = new Date().getTime()
let errMsg = null

async function sendToSlack(error) {
  if (SLACK_WEBHOOK_URL) {
    const webhook = new IncomingWebhook(SLACK_WEBHOOK_URL)
    const text = ` :fire: _An error has occurred at_ \`${os.hostname()}\`\n \`\`\`${JSON.stringify(
      error[1]
    )} \`\`\`\n>*System information*\n>*memory*: ${os.freemem()}/${os.totalmem()}\n>*machine*: ${os.machine()}\n>*platform*: ${os.platform()}\n>*upTime*: ${os.uptime()}\n>*version*: ${os.version()}
   `
    try {
      if (errMsg == error[1].message) {
        const currentDate = new Date()
        const oneMinuteAgo = new Date(currentDate.getTime() - 60000)
        if (slackSentTime < oneMinuteAgo.getTime()) {
          await webhook.send({ text })
          errMsg = error[1].message
          slackSentTime = new Date().getTime()
        }
      } else {
        await webhook.send({ text })
        errMsg = error[1].message
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

export function buildReporterJobId({
  aggregatorAddress,
  roundId,
  deploymentName
}: {
  aggregatorAddress: string
  roundId: number
  deploymentName: string
}) {
  return `${roundId}-${aggregatorAddress}-${deploymentName}`
}
