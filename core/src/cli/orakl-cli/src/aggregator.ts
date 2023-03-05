import axios from 'axios'
import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove,
  buildUrl
} from './utils'
import { computeDataHash } from './utils'
import { ReadFile } from './cli-types'
import { IAggregator } from './types'
import { CliError, CliErrorCode } from './error'
import { ORAKL_NETWORK_API_URL } from './settings'

const AGGREGATOR_ENDPOINT = buildUrl(ORAKL_NETWORK_API_URL, 'aggregator')

export function aggregatorSub(db) {
  // aggregator list [--active] [--chain [chain]]
  // aggregator insert --file-path [file-path] --chain [chain]
  // aggregator remove --id [id]

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
  const insertFromChain = command({
    name: 'insertFromChain',
    args: {
      aggregatorId: option({ type: cmdstring, long: 'aggregator-id' }),
      adapter: option({ type: cmdstring, long: 'adapter' }),
      fromChain: option({ type: cmdstring, long: 'from-chain' }),
      toChain: option({ type: cmdstring, long: 'to-chain' }),
      dryrun: dryrunOption
    },
    handler: insertFromChainHandler(db)
  })

  return subcommands({
    name: 'aggregator',
    cmds: { list, insert, remove, insertFromChain }
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

export function insertFromChainHandler(db) {
  async function wrapper({
    aggregatorId,
    adapter,
    fromChain,
    toChain,
    dryrun
  }: {
    aggregatorId: string
    adapter: string
    fromChain: string
    toChain: string
    dryrun?: boolean
  }) {
    const fromChainId = await chainToId(db, fromChain)
    const toChainId = await chainToId(db, toChain)

    const queryAdapter = `SELECT id from Adapter WHERE adapterId='${adapter}' and chainId=${fromChainId};`
    const result = await db.get(queryAdapter)
    const adapterId = result.id
    const query = `INSERT INTO Aggregator (chainId, aggregatorId, adapterId, data)
    SELECT ${toChainId}, aggregatorId, adapterId, data FROM Aggregator
    WHERE chainId=${fromChainId} and aggregatorId='${aggregatorId}' and adapterId='${adapterId}'`

    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultInsert(result))
    }
  }
  return wrapper
}
