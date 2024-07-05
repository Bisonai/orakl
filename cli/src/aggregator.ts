import axios from 'axios'
import {
  boolean as cmdboolean,
  command,
  flag,
  option,
  string as cmdstring,
  subcommands,
} from 'cmd-ts'
import { IAggregator, ReadFile } from './cli-types.js'
import { ORAKL_NETWORK_API_URL, WORKER_SERVICE_HOST, WORKER_SERVICE_PORT } from './settings.js'
import {
  buildUrl,
  chainOptionalOption,
  fetcherTypeOptionalOption,
  idOption,
  isOraklNetworkApiHealthy,
  isServiceHealthy,
} from './utils.js'

const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')

export function aggregatorSub() {
  // aggregator list [--active] [--chain ${chain}]
  // aggregator insert --source ${source} --chain ${chain}
  // aggregator remove --id ${id}
  // aggregator hash --source ${source} --verify
  // aggregator active --host ${host} --port ${port}
  // aggregator activate --host ${host} --port ${port} --aggregatorHash ${aggregatorHash}
  // aggregator deactivate --host ${host} --port ${port} --aggregatorHash ${aggregatorHash}

  const list = command({
    name: 'list',
    args: {
      active: flag({
        long: 'active',
      }),
      chain: chainOptionalOption,
    },
    handler: listHandler(true),
  })

  const insert = command({
    name: 'insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source',
      }),
      chain: option({
        type: cmdstring,
        long: 'chain',
      }),
      fetcherType: fetcherTypeOptionalOption,
    },
    handler: insertHandler(),
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
    },
    handler: removeHandler(),
  })

  const hash = command({
    name: 'hash',
    args: {
      verify: flag({
        type: cmdboolean,
        long: 'verify',
      }),
      data: option({
        type: ReadFile,
        long: 'source',
      }),
    },
    handler: hashHandler(),
  })

  const active = command({
    name: 'active',
    args: {
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => WORKER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(WORKER_SERVICE_PORT),
      }),
    },
    handler: activeHandler(),
  })

  const activate = command({
    name: 'activate',
    args: {
      aggregatorHash: option({
        type: cmdstring,
        long: 'aggregatorHash',
      }),
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => WORKER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(WORKER_SERVICE_PORT),
      }),
    },
    handler: activateHandler(),
  })

  const deactivate = command({
    name: 'deactivate',

    args: {
      aggregatorHash: option({
        type: cmdstring,
        long: 'aggregatorHash',
      }),
      host: option({
        type: cmdstring,
        long: 'host',
        defaultValue: () => WORKER_SERVICE_HOST,
      }),
      port: option({
        type: cmdstring,
        long: 'port',
        defaultValue: () => String(WORKER_SERVICE_PORT),
      }),
    },
    handler: deactivateHandler(),
  })

  return subcommands({
    name: 'aggregator',
    cmds: { list, insert, remove, hash, active, activate, deactivate },
  })
}

export function listHandler(print?: boolean) {
  async function wrapper({ chain, active }: { chain?: string; active?: boolean }) {
    if (!(await isOraklNetworkApiHealthy())) return

    // cmd-ts does not allow to set boolean flag to undefined. It can
    // be either true of false. When `active` is not set, we assume that
    // user wants to see all aggregators.
    if (!active) {
      active = undefined
    }

    try {
      const url = new URL(AGGREGATOR_ENDPOINT)
      if (active) {
        url.searchParams.append('active', 'true')
      }
      if (chain) {
        url.searchParams.append('chain', chain)
      }

      const result = (await axios.get(url.toString())).data
      if (print) {
        console.dir(result, { depth: null })
      }
      return result
    } catch (e) {
      console.dir(e?.response?.data, { depth: null })
    }
  }
  return wrapper
}

export function insertHandler() {
  async function wrapper({
    data,
    chain,
    fetcherType,
  }: {
    data
    chain: string
    fetcherType?: number
  }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.post(AGGREGATOR_ENDPOINT, { ...data, chain, fetcherType })).data
      console.dir(result, { depth: null })
      return result
    } catch (e) {
      console.error('Aggregator was not inserted. Reason:')
      console.error(e?.response?.data)
      return e?.response?.data
    }
  }
  return wrapper
}

export function removeHandler() {
  async function wrapper({ id }: { id: number }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const endpoint = buildUrl(AGGREGATOR_ENDPOINT, id.toString())
      const result = (await axios.delete(endpoint)).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Aggregator was not deleted. Reason:')
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function hashHandler() {
  async function wrapper({ data, verify }: { data; verify: boolean }) {
    try {
      const endpoint = buildUrl(AGGREGATOR_ENDPOINT, 'hash')
      const aggregator = data as IAggregator
      const aggregatorWithCorrectHash = (
        await axios.post(endpoint, aggregator, {
          params: { verify },
        })
      ).data
      console.dir(aggregatorWithCorrectHash, { depth: null })
      return aggregatorWithCorrectHash
    } catch (e) {
      console.error('Aggregator hash could not be computed. Reason:')
      const errMsg = e?.response?.data ? e.response.data : e.message

      console.error(errMsg)
      return errMsg
    }
  }
  return wrapper
}

export function activeHandler() {
  async function wrapper({ host, port }: { host: string; port: string }) {
    const aggregatorServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(aggregatorServiceEndpoint))) return

    const activeAggregatorEndpoint = buildUrl(aggregatorServiceEndpoint, 'active')

    try {
      const result = (await axios.get(activeAggregatorEndpoint)).data
      console.log(result)
    } catch (e) {
      console.error(e?.response?.data?.message)
    }
  }
  return wrapper
}

export function activateHandler() {
  async function wrapper({
    host,
    port,
    aggregatorHash,
  }: {
    host: string
    port: string
    aggregatorHash: string
  }) {
    const aggregatorServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(aggregatorServiceEndpoint))) return

    const activateAggregatorEndpoint = buildUrl(
      aggregatorServiceEndpoint,
      `activate/${aggregatorHash}`,
    )

    try {
      const result = (await axios.get(activateAggregatorEndpoint)).data
      console.log(result?.message)
    } catch (e) {
      console.error('Aggregator was not activated. Reason:')
      console.error(e?.response?.data?.message)
      throw e
    }
  }
  return wrapper
}

export function deactivateHandler() {
  async function wrapper({
    host,
    port,
    aggregatorHash,
  }: {
    host: string
    port: string
    aggregatorHash: string
  }) {
    const aggregatorServiceEndpoint = `${host}:${port}`
    if (!(await isServiceHealthy(aggregatorServiceEndpoint))) return

    const deactivateAggregatorEndpoint = buildUrl(
      aggregatorServiceEndpoint,
      `deactivate/${aggregatorHash}`,
    )

    try {
      const result = (await axios.get(deactivateAggregatorEndpoint)).data
      console.log(result?.message)
    } catch (e) {
      console.error('Aggregator was not deactivated. Reason:')
      console.error(e?.response?.data?.message)
      throw e
    }
  }
  return wrapper
}
