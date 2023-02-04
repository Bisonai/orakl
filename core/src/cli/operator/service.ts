import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { Logger } from 'pino'
import { dryrunOption, idOption, formatResultInsert, formatResultRemove } from './utils'

export function serviceSub(db, logger: Logger) {
  // service list
  // service insert --name [name] [--dryrun]
  // service remove --id [id]     [--dryrun]

  const list = command({
    name: 'list',
    args: {},
    handler: listHandler(db, true, logger)
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
    name: 'service',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
  async function wrapper() {
    const query = 'SELECT * FROM Service'
    const result = await db.all(query)
    if (print) {
      logger?.info(result)
    }
    return result
  }
  return wrapper
}

export function insertHandler(db, logger?: Logger) {
  async function wrapper({ name, dryrun }: { name: string; dryrun?: boolean }) {
    const query = `INSERT INTO Service (name) VALUES ('${name}')`
    if (dryrun) {
      logger?.info(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultInsert(result))
    }
  }
  return wrapper
}

export function removeHandler(db, logger?: Logger) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM Service WHERE id=${id}`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}
