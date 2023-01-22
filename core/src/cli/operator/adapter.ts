import { flag, command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  chainOptionalOption,
  chainToId,
  dryrunOption,
  idOption,
  formatResultInsert,
  formatResultRemove
} from './utils'
import { computeDataHash } from '../utils'
import { printObject } from '../../utils'
import { ReadFile } from './types'

export function adapterSub(db) {
  // adapter list
  // adapter insert --file-path [file-path] [--dryrun]
  // adapter remove --id [id]               [--dryrun]

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
    name: 'adapter',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?) {
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
          console.log(`ID: ${r.id}`)
          printObject(rJson)
        }
      }
    }
    return result
  }
  return wrapper
}

export function insertHandler(db) {
  async function wrapper({ data, chain, dryrun }: { data; chain: string; dryrun?: boolean }) {
    const chainId = await chainToId(db, chain)
    const adapterObject = await computeDataHash({ data })
    const adapter = JSON.stringify(adapterObject)
    const query = `INSERT INTO Adapter (chainId, adapterId, data) VALUES (${chainId}, '${adapterObject.id}', '${adapter}')`

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
    const query = `DELETE FROM Adapter WHERE id=${id}`
    if (dryrun) {
      console.debug(query)
    } else {
      const result = await db.run(query)
      console.log(formatResultRemove(result))
    }
  }
  return wrapper
}
