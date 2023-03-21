import axios from 'axios'
import {
  flag,
  command,
  subcommands,
  option,
  string as cmdstring,
  boolean as cmdboolean
} from 'cmd-ts'
import { chainOptionalOption, idOption, buildUrl, computeAggregatorHash } from './utils'
import { ReadFile, IAggregator } from './cli-types'
import { ORAKL_NETWORK_API_URL } from './settings'

const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'api/v1/aggregator')

export function aggregatorSub() {
  // aggregator list [--active] [--chain [chain]]
  // aggregator insert --file-path [file-path] --chain [chain]
  // aggregator remove --id [id]
  // aggregator hash --file-path [file-path] --verify

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
        long: 'file-path'
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
        long: 'file-path'
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
    // cmd-ts does not allow to set boolean flag to undefined. It can
    // be either true of false. When `active` is not set, we assume that
    // user wants to see all aggregators.
    if (!active) {
      active = undefined
    }
    const result = (await axios.get(AGGREGATOR_ENDPOINT, { data: { chain, active } })).data
    if (print) {
      console.dir(result, { depth: null })
    }
    return result
  }
  return wrapper
}

export function insertHandler() {
  async function wrapper({ data, chain }: { data; chain: string }) {
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
    const endpoint = buildUrl(AGGREGATOR_ENDPOINT, id.toString())
    const result = (await axios.delete(endpoint)).data
    console.dir(result, { depth: null })
  }
  return wrapper
}

export function hashHandler() {
  async function wrapper({ data, verify }: { data; verify: boolean }) {
    try {
      const aggregator = data as IAggregator
      const aggregatorWithCorrectHash = await computeAggregatorHash({ data: aggregator, verify })
      console.dir(aggregatorWithCorrectHash, { depth: null })
    } catch (e) {
      console.error(e.message)
    }
  }
  return wrapper
}
