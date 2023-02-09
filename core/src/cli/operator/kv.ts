import { command, subcommands, optional, option, string as cmdstring } from 'cmd-ts'
import { Logger } from 'pino'
import {
  dryrunOption,
  chainOptionalOption,
  chainToId,
  formatResultInsert,
  formatResultRemove
} from './utils'
import { ReadFile } from './types'

export function kvSub(db, logger: Logger) {
  // kv list        [--chain [chain]] [--key [key]]
  // kv insert       --chain [chain]   --key [key] --value [value] [--dryrun]
  // kv remove       --chain [chain]   --key [key]                 [--dryrun]
  // kv update       --chain [chain]   --key [key] --value [value] [--dryrun]
  // kv insertMany   --chain [chain] --file-path [file-path]       [--dryrun]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      key: option({
        type: optional(cmdstring),
        long: 'key'
      })
    },
    handler: listHandler(db, true, logger)
  })

  const insert = command({
    name: 'insert',
    args: {
      key: option({
        type: cmdstring,
        long: 'key'
      }),
      value: option({
        type: cmdstring,
        long: 'value',
        defaultValue: () => ''
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      dryrun: dryrunOption
    },
    handler: insertHandler(db, logger)
  })

  const insertMany = command({
    name: 'insertMany',
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
    handler: insertManyHandler(db, logger)
  })

  const remove = command({
    name: 'remove',
    args: {
      key: option({
        type: cmdstring,
        long: 'key'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      dryrun: dryrunOption
    },
    handler: removeHandler(db, logger)
  })

  const update = command({
    name: 'update',
    args: {
      key: option({
        type: cmdstring,
        long: 'key'
      }),
      value: option({
        type: cmdstring,
        long: 'value',
        defaultValue: () => ''
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      dryrun: dryrunOption
    },
    handler: updateHandler(db, logger)
  })

  return subcommands({
    name: 'kv',
    cmds: { list, insert, insertMany, remove, update }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
  async function wrapper({ chain, key }: { chain?: string; key?: string }) {
    let where = ''
    if (chain) {
      const chainId = await chainToId(db, chain)
      where += `WHERE chainId=${chainId}`
    }

    if (key) {
      if (where) {
        where += ` AND `
      } else {
        where = `WHERE `
      }
      where += `key='${key}'`
    }
    const query = `SELECT * FROM Kv ${where};`
    const result = await db.all(query)
    if (print) {
      logger?.info(result)
    }
    return result
  }
  return wrapper
}

export function insertHandler(db, logger?: Logger) {
  async function wrapper({
    key,
    value,
    chain,
    dryrun
  }: {
    key: string
    value: string
    chain: string
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const query = `INSERT INTO Kv (chainId, key, value) VALUES (${chainId}, '${key}', '${value}');`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultInsert(result))
    }
  }
  return wrapper
}

export function insertManyHandler(db, logger?: Logger) {
  async function wrapper({ data, chain, dryrun }: { data; chain: string; dryrun?: boolean }) {
    const chainId = await chainToId(db, chain)

    const values: string[] = []
    for (const key in data) {
      values.push(`(${chainId}, '${key}', '${data[key]}')`)
    }

    const query = `INSERT INTO Kv (chainId, key, value) VALUES ${values.join()};`
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
  async function wrapper({ key, chain, dryrun }: { key: string; chain: string; dryrun?: boolean }) {
    const chainId = await chainToId(db, chain)
    const query = `DELETE FROM Kv WHERE chainId=${chainId} AND key='${key}';`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}

export function updateHandler(db, logger?: Logger) {
  async function wrapper({
    key,
    value,
    chain,
    dryrun
  }: {
    key: string
    value: string
    chain: string
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const query = `UPDATE Kv SET value='${value}' WHERE chainId=${chainId} AND key='${key}';`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(result)
    }
  }
  return wrapper
}
