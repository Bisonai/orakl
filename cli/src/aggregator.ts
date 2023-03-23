import axios from 'axios'
import {
  flag,
  command,
  subcommands,
  option,
  string as cmdstring,
  boolean as cmdboolean
} from 'cmd-ts'
import { chainOptionalOption, idOption, buildUrl, isOraklNetworkApiHealthy } from './utils'
import { ReadFile, IAggregator } from './cli-types'
import { ORAKL_NETWORK_API_URL } from './settings'

const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')

export function aggregatorSub() {
  // aggregator list [--active] [--chain ${chain}]
  // aggregator insert --source ${source} --chain ${chain}
  // aggregator remove --id ${id}
  // aggregator hash --source ${source} --verify

  const list = command({
    name: 'list',
    args: {
      active: flag({
        long: 'active'
      }),
      chain: chainOptionalOption
    },
    handler: listHandler(true)
  })

  const insert = command({
    name: 'insert',
    args: {
      data: option({
        type: ReadFile,
        long: 'source'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      })
    },
    handler: insertHandler()
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption
    },
    handler: removeHandler()
  })

  const hash = command({
    name: 'hash',
    args: {
      verify: flag({
        type: cmdboolean,
        long: 'verify'
      }),
      data: option({
        type: ReadFile,
        long: 'source'
      })
    },
    handler: hashHandler()
  })

  return subcommands({
    name: 'aggregator',
    cmds: { list, insert, remove, hash }
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
      const result = (await axios.get(AGGREGATOR_ENDPOINT, { data: { chain, active } })).data
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
  async function wrapper({ data, chain }: { data; chain: string }) {
    if (!(await isOraklNetworkApiHealthy())) return

    try {
      const result = (await axios.post(AGGREGATOR_ENDPOINT, { ...data, chain })).data
      console.dir(result, { depth: null })
    } catch (e) {
      console.error('Aggregator was not inserted. Reason:')
      console.error(e?.response?.data?.message)
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
          params: { verify }
        })
      ).data
      console.dir(aggregatorWithCorrectHash, { depth: null })
    } catch (e) {
      console.error('Aggregator hash could not be computed. Reason:')
      console.error(e.message)
    }
  }
  return wrapper
}
