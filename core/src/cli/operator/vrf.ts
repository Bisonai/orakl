import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  dryrunOption,
  idOption,
  chainOptionalOption,
  chainToId,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function vrfSub(db) {
  // vrf list   [--chain [chain]]                                                [--dryrun]
  // vrf insert  --chain [chain] --pk [pk] --sk [sk] --pk_x [pk_x] --pk_y [pk_y] [--dryrun]
  // vrf remove  --id [id]                                                       [--dryrun]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      dryrun: dryrunOption
    },
    handler: listHandler(db, true)
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

  return subcommands({
    name: 'vrf',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?) {
  async function wrapper({ chain, dryrun }: { chain?: string; dryrun?: boolean }) {
    let where = ''
    if (chain) {
      where += `AND Chain.name = '${chain}'`
    }
    const query = `SELECT VrfKey.id, Chain.name as chain, sk, pk, pk_x, pk_y FROM VrfKey INNER JOIN Chain
   ON VrfKey.chainId = Chain.id ${where};`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.all(query)
      console.log(result)
      return result
    }
  }
  return wrapper
}

export function insertHandler(db) {
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
    const query = `DELETE FROM VrfKey WHERE id=${id};`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}
