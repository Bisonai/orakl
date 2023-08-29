import axios from 'axios'
import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { buildUrl, isOraklFetcherHealthy } from './utils'

export function fetcherSub() {
  // fetcher active --host ${host} --port ${port}
  // fetcher start --id ${aggregatorHash} --chain ${chain} --host ${host} --port ${port}
  // fetcher stop --id ${aggregatorHash} --chain ${chain}  --host ${host} --port ${port}

  const active = command({
    name: 'active',
    args: {
      host: option({
        type: cmdstring,
        long: 'host'
      }),
      port: option({
        type: cmdstring,
        long: 'port'
      })
    },
    handler: activeHandler()
  })

  const start = command({
    name: 'start',
    args: {
      id: option({
        type: cmdstring,
        long: 'id'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      host: option({
        type: cmdstring,
        long: 'host'
      }),
      port: option({
        type: cmdstring,
        long: 'port'
      })
    },
    handler: startHandler()
  })

  const stop = command({
    name: 'stop',
    args: {
      id: option({
        type: cmdstring,
        long: 'id'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      host: option({
        type: cmdstring,
        long: 'host'
      }),
      port: option({
        type: cmdstring,
        long: 'port'
      })
    },
    handler: stopHandler()
  })

  return subcommands({
    name: 'fetcher',
    cmds: { active, start, stop }
  })
}

export function activeHandler() {
  async function wrapper({ host, port }: { host: string; port: string }) {
    const FetcherEndpoint = `${host}:${port}/api/v1`
    if (!(await isOraklFetcherHealthy(FetcherEndpoint))) return

    try {
      const activeFetcherEndpoint = buildUrl(FetcherEndpoint, 'active')
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
    port
  }: {
    id: string
    chain: string
    host: string
    port: string
  }) {
    const fetcherEndpoint = `${host}:${port}/api/v1`
    if (!(await isOraklFetcherHealthy(fetcherEndpoint))) return

    try {
      const endpoint = buildUrl(fetcherEndpoint, `start/${id}`)
      const result = (await axios.get(endpoint, { data: { chain } })).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}

export function stopHandler() {
  async function wrapper({
    id,
    chain,
    host,
    port
  }: {
    id: string
    chain: string
    host: string
    port: string
  }) {
    const fetcherEndpoint = `${host}:${port}/api/v1`
    console.log('fetcherEndPoint:', fetcherEndpoint)
    if (!(await isOraklFetcherHealthy(fetcherEndpoint))) return

    try {
      const endpoint = buildUrl(fetcherEndpoint, `stop/${id}`)
      const result = (await axios.get(endpoint, { data: { chain } })).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}
