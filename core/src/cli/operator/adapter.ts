import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function adapterSub(db) {
  // chain list
  // chain insert --name [name] [--dryrun]
  // chain remove --id [id]     [--dryrun]

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

  //   const insert = command({
  //     name: 'insert',
  //     args: {
  //       name: option({
  //         type: cmdstring,
  //         long: 'name'
  //       }),
  //       dryrun: dryrunOption
  //     },
  //     handler: insertHandler(db)
  //   })

  //   const remove = command({
  //     name: 'remove',
  //     args: {
  //       id: idOption,
  //       dryrun: dryrunOption
  //     },
  //     handler: removeHandler(db)
  //   })
  //
  return subcommands({
    name: 'adapter',
    cmds: { list /*, insert, remove*/ }
  })
}

export function listHandler(db, print?) {
  async function wrapper({ chain, active }: { chain?: string; active?: boolean }) {
    let where = ''
    if (chain) {
      const chainId = await chainToId(db, chain)
      where += ` WHERE chainId=${chainId}`
    }
    const query = `SELECT data FROM Adapter ${where};`
    const result = await db.all(query)
    if (print) {
      for (const r of result) {
        const rJson = JSON.parse(r.data)
        if (!active || rJson.active) {
          console.dir(rJson)
        }
      }
    }
    return result
  }
  return wrapper
}
