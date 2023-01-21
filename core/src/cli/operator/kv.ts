import { command, subcommands, optional, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  dryrunOption,
  chainToId,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function kvCmd(db) {
  // kv list   [--chain [chain]] [--key [key]]
  // kv insert  --chain [chain]   --key [key] --value [value] [--dryrun]
  // kv remove  --chain [chain]   --key [key]                 [--dryrun]
  // kv update  --chain [chain]   --key [key] --value [value] [--dryrun]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      key: option({
        type: optional(cmdstring),
        long: 'key'
      })
    },
    handler: listHandler(db)
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
        long: 'value'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      dryrun: dryrunOption
    },
    handler: insertHandler(db)
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
    handler: removeHandler(db)
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
        long: 'value'
      }),
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      dryrun: dryrunOption
    },
    handler: updateHandler(db)
  })

  return subcommands({
    name: 'kv',
    cmds: { list, insert, remove, update }
  })
}

export function listHandler(db) {
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
    console.log(result)
    return result
  }
  return wrapper
}

export function insertHandler(db) {
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
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultInsert(result))
    }
  }
  return wrapper
}

export function removeHandler(db) {
  async function wrapper({ key, chain, dryrun }: { key: string; chain: string; dryrun?: boolean }) {
    const chainId = await chainToId(db, chain)
    const query = `DELETE FROM Kv WHERE chainId=${chainId} AND key='${key}';`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}

export function updateHandler(db) {
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
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(result)
    }
  }
  return wrapper
}
