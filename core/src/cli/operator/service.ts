import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { dryrunOption, idOption, formatResultInsert, formatResultRemove } from './utils'

export function serviceSub(db) {
  // service list
  // service insert --name [name] [--dryrun]
  // service remove --id [id]     [--dryrun]

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
    name: 'service',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?) {
  async function wrapper() {
    const query = 'SELECT * FROM Service'
    const result = await db.all(query)
    if (print) {
      console.log(result)
    }
    return result
  }
  return wrapper
}

export function insertHandler(db) {
  async function wrapper({ name, dryrun }: { name: string; dryrun?: boolean }) {
    const query = `INSERT INTO Service (name) VALUES ('${name}')`
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
    const query = `DELETE FROM Service WHERE id=${id}`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}
