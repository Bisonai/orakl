import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { Logger } from 'pino'
import {
  dryrunOption,
  idOption,
  chainOptionalOption,
  chainToId,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function vrfSub(db, logger: Logger) {
  // vrf list   [--chain [chain]]                                                [--dryrun]
  // vrf insert  --chain [chain] --pk [pk] --sk [sk] --pk_x [pk_x] --pk_y [pk_y] [--dryrun]
  // vrf remove  --id [id]                                                       [--dryrun]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      dryrun: dryrunOption
    },
    handler: listHandler(db, true, logger)
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      sk: option({
        type: cmdstring,
        long: 'sk'
      }),
      pk: option({
        type: cmdstring,
        long: 'pk'
      }),
      pk_x: option({
        type: cmdstring,
        long: 'pk_x'
      }),
      pk_y: option({
        type: cmdstring,
        long: 'pk_y'
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
    name: 'vrf',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
  async function wrapper({ chain, dryrun }: { chain?: string; dryrun?: boolean }) {
    let where = ''
    if (chain) {
      where += `AND Chain.name = '${chain}'`
    }
    const query = `SELECT VrfKey.id, Chain.name as chain, sk, pk, pk_x, pk_y FROM VrfKey INNER JOIN Chain
   ON VrfKey.chainId = Chain.id ${where};`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.all(query)
      if (print) {
        logger?.info(result)
      }
      return result
    }
  }
  return wrapper
}

export function insertHandler(db, logger?: Logger) {
  async function wrapper({
    chain,
    pk,
    sk,
    pk_x,
    pk_y,
    dryrun
  }: {
    chain: string
    pk: string
    sk: string
    pk_x: string
    pk_y: string
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const query = `INSERT INTO VrfKey (chainId, sk, pk, pk_x, pk_y) VALUES (${chainId}, '${sk}', '${pk}', '${pk_x}', '${pk_y}');`
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
    const query = `DELETE FROM VrfKey WHERE id=${id};`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}
