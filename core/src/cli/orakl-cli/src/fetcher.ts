import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { idOption, buildUrl } from './utils'
import { ORAKL_NETWORK_FETCHER_URL } from './settings'
import { CliError, CliErrorCode } from './errors'

export function fetcherSub() {
  const start = command({
    name: 'start',
    args: {},
    handler: startHandler()
  })

  const stop = command({
    name: 'stop',
    args: {},
    handler: stopHandler()
  })

  return subcommands({
    name: 'fetcher',
    cmds: { start, stop }
  })
}

export function startHandler() {
  async function wrapper() {
    const startEndpoint = buildUrl(ORAKL_NETWORK_FETCHER_URL, 'start')
    console.log('start')
  }
  return wrapper
}

export function stopHandler() {
  async function wrapper() {
    const stopEndpoint = buildUrl(ORAKL_NETWORK_FETCHER_URL, 'stop')
    console.log('stop')
  }
  return wrapper
}
