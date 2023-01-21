import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { dryrunOption, idOption, formatResultInsert, formatResultRemove } from './utils'

export function chainSub(db) {
  // chain list
  // chain insert --name [name] [--dryrun]
  // chain remove --id [id]     [--dryrun]

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(db)
  })

  const insert = command({
    name: 'insert',
    args: {
      name: option({
        type: cmdstring,
        long: 'name'
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
    name: 'chain',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db) {
  async function wrapper() {
    const query = 'SELECT * FROM Chain'
    const result = await db.all(query)
    console.log(result)
    return result
  }
  return wrapper
}

export function insertHandler(db) {
  async function wrapper({ name, dryrun }: { name: string; dryrun?: boolean }) {
    const query = `INSERT INTO Chain (name) VALUES ('${name}')`
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
    const query = `DELETE FROM Chain WHERE id=${id}`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}
