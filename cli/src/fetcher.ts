import axios from 'axios'
import { command, option, string as cmdstring, subcommands } from 'cmd-ts'
import { FETCHER_API_VERSION, FETCHER_HOST, FETCHER_PORT } from './settings.js'
import { buildFetcherUrl, buildUrl, isOraklFetcherHealthy } from './utils.js'

export function fetcherSub() {
  // fetcher active --host ${host} --port ${port}
  // fetcher start --id ${aggregatorHash} --chain ${chain} --host ${host} --port ${port}
  // fetcher stop --id ${aggregatorHash} --chain ${chain}  --host ${host} --port ${port}

  const active = command({
    name: 'active',
    args: {
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => FETCHER_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(FETCHER_PORT),
      }),
    },
    handler: activeHandler(),
  })

  const start = command({
    name: 'start',
    args: {
      id: option({
        type: cmdstring,
        long: 'id',
      }),
      chain: option({
        type: cmdstring,
        long: 'chain',
      }),
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => FETCHER_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(FETCHER_PORT),
      }),
    },
    handler: startHandler(),
  })

  const stop = command({
    name: 'stop',
    args: {
      id: option({
        type: cmdstring,
        long: 'id',
      }),
      chain: option({
        type: cmdstring,
        long: 'chain',
      }),
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => FETCHER_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(FETCHER_PORT),
      }),
    },
    handler: stopHandler(),
  })

  return subcommands({
    name: 'fetcher',
    cmds: { active, start, stop },
  })
}

export function activeHandler() {
  async function wrapper({ host, port }: { host: string; port: string }) {
    const fetcherEndpoint = buildFetcherUrl(host, port, FETCHER_API_VERSION)
    if (!(await isOraklFetcherHealthy(fetcherEndpoint))) return

    try {
      const activeFetcherEndpoint = buildUrl(fetcherEndpoint, 'active')
      const result = (await axios.get(activeFetcherEndpoint)).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function startHandler() {
  async function wrapper({
    id,
    chain,
    host,
    port,
  }: {
    id: string
    chain: string
    host: string
    port: string
  }) {
    const fetcherEndpoint = buildFetcherUrl(host, port, FETCHER_API_VERSION)
    if (!(await isOraklFetcherHealthy(fetcherEndpoint))) return

    try {
      const endpoint = buildUrl(fetcherEndpoint, `start/${id}`)
      const result = (await axios.get(endpoint, { data: { chain } })).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data, { depth: null })
      throw e
    }
  }
  return wrapper
}

export function stopHandler() {
  async function wrapper({
    id,
    chain,
    host,
    port,
  }: {
    id: string
    chain: string
    host: string
    port: string
  }) {
    const fetcherEndpoint = buildFetcherUrl(host, port, FETCHER_API_VERSION)
    if (!(await isOraklFetcherHealthy(fetcherEndpoint))) return

    try {
      const endpoint = buildUrl(fetcherEndpoint, `stop/${id}`)
      const result = (await axios.get(endpoint, { data: { chain } })).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data?.message)
      throw e
    }
  }
  return wrapper
}
