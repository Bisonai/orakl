import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove
} from './utils'
import { computeDataHash } from './utils'
import { ReadFile } from './cli-types'
import { IAggregator } from './types'
import { CliError, CliErrorCode } from './error'

export function aggregatorSub(db) {
  // aggregator list [--active] [--chain [chain]]
  // aggregator insert --file-path [file-path] --adapter [adapter] --chain [chain] [--dryrun]
  // aggregator remove --id [id]                                                   [--dryrun]

  const list = command({
    name: 'list',
    args: {
      active: flag({
        long: 'active'
      }),
      chain: chainOptionalOption
    },
    handler: listHandler(db, true)
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
      }),
      adapter: option({
        type: cmdstring,
        long: 'adapter'
      }),
      dryrun: dryrunOption
    },
    handler: insertHandler(db)
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: removeHandler(db)
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

export function listHandler(db, print?: boolean) {
  async function wrapper({ chain, active }: { chain?: string; active?: boolean }) {
    let where = ''
    if (chain) {
      const chainId = await chainToId(db, chain)
      where += ` WHERE chainId=${chainId}`
    }
    const query = `SELECT id, aggregatorId, adapterId, data FROM Aggregator ${where};`
    const result = await db.all(query)
    if (print) {
      for (const r of result) {
        const rJson = JSON.parse(r.data)
        if (!active || rJson.active) {
          console.log(`ID: ${r.id}`)
          console.log(rJson)
        }
      }
    }
    return result
  }
  return wrapper
}

export function insertHandler(db) {
  async function wrapper({
    data,
    chain,
    adapter,
    dryrun
  }: {
    data
    chain: string
    adapter
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const aggregatorObject = (await computeDataHash({ data })) as IAggregator
    const aggregator = JSON.stringify(aggregatorObject)

    let adapterId
    if (adapter != aggregatorObject.adapterId) {
      throw new CliError(CliErrorCode.InconsistentAdapterId)
    } else {
      const query = `SELECT id from Adapter WHERE adapterId='${adapter}';`
      const result = await db.get(query)
      adapterId = result.id
    }
    const query = `INSERT INTO Aggregator (chainId, aggregatorId, adapterId, data) VALUES (${chainId}, '${aggregatorObject.id}', ${adapterId}, '${aggregator}')`

    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultInsert(result))
    }
  }
  return wrapper
}

export function removeHandler(db) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM Aggregator WHERE id=${id}`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
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
