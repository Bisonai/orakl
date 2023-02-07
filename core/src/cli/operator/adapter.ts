import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { Logger } from 'pino'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove
} from './utils'
import { computeDataHash } from '../utils'
import { ReadFile } from './types'

export function adapterSub(db, logger: Logger) {
  // adapter list [--active] [--chain [chain]]
  // adapter insert --file-path [file-path] --chain [chain] [--dryrun]
  // adapter remove --id [id]                               [--dryrun]

  const list = command({
    name: 'list',
    args: {
      active: flag({
        long: 'active'
      }),
      chain: chainOptionalOption
    },
    handler: listHandler(db, true, logger)
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
      dryrun: dryrunOption
    },
    handler: insertHandler(db, logger)
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: removeHandler(db, logger)
  })

  const insertFromChain = command({
    name: 'insertFromChain',
    args: {
      adapterId: option({ type: cmdstring, long: 'adapter-id' }),
      fromChain: option({ type: cmdstring, long: 'from-chain' }),
      toChain: option({ type: cmdstring, long: 'to-chain' }),
      dryrun: dryrunOption
    },
    handler: insertFromChainHandler(db, logger)
  })

  return subcommands({
    name: 'adapter',
    cmds: { list, insert, remove, insertFromChain }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
  async function wrapper({ chain, active }: { chain?: string; active?: boolean }) {
    let where = ''
    if (chain) {
      const chainId = await chainToId(db, chain)
      where += ` WHERE chainId=${chainId}`
    }
    const query = `SELECT id, data FROM Adapter ${where};`
    const result = await db.all(query)
    if (print) {
      for (const r of result) {
        const rJson = JSON.parse(r.data)
        if (!active || rJson.active) {
          logger?.info(`ID: ${r.id}`)
          logger?.info(rJson)
        }
      }
    }
    return result
  }
  return wrapper
}

export function insertHandler(db, logger?: Logger) {
  async function wrapper({ data, chain, dryrun }: { data; chain: string; dryrun?: boolean }) {
    const chainId = await chainToId(db, chain)
    const adapterObject = await computeDataHash({ data })
    const adapter = JSON.stringify(adapterObject)
    const query = `INSERT INTO Adapter (chainId, adapterId, data) VALUES (${chainId}, '${adapterObject.id}', '${adapter}')`

    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultInsert(result))
    }
  }
  return wrapper
}

export function removeHandler(db, logger?: Logger) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM Adapter WHERE id=${id}`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}

export function insertFromChainHandler(db, logger?: Logger) {
  async function wrapper({
    adapterId,
    fromChain,
    toChain,
    dryrun
  }: {
    adapterId: string
    fromChain: string
    toChain: string
    dryrun?: boolean
  }) {
    const fromChainId = await chainToId(db, fromChain)
    const toChainId = await chainToId(db, toChain)

    const query = `INSERT INTO Adapter (chainId, adapterId, data) SELECT ${toChainId},adapterId, data FROM Adapter WHERE chainId=${fromChainId} and adapterId='${adapterId}'`

    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultInsert(result))
    }
  }
  return wrapper
}
