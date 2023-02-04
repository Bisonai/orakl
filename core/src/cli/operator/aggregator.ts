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
import { IAggregator } from '../..//types'
import { CliError, CliErrorCode } from './error'

export function aggregatorSub(db, logger: Logger) {
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
      adapter: option({
        type: cmdstring,
        long: 'adapter'
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

  return subcommands({
    name: 'aggregator',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
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
    const query = `DELETE FROM Aggregator WHERE id=${id}`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}
