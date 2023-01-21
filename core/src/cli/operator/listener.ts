import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import {
  dryrunOption,
  idOption,
  chainOptionalOption,
  serviceOptionalOption,
  chainToId,
  serviceToId
} from './utils'

export function listenerSub(db) {
  // listener list   [--chain [chain]] [--service [service]]                                            [--dryrun]
  // listener insert  --chain [chain]   --service [service] --address [address] --eventName [eventName] [--dryrun]
  // listener remove  --id [id]                                                                         [--dryrun]

  const list = command({
    name: 'list',
    args: {
      chain: chainOptionalOption,
      service: serviceOptionalOption,
      dryrun: dryrunOption
    },
    handler: listHandler(db)
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: option({
        type: cmdstring,
        long: 'chain'
      }),
      service: option({
        type: cmdstring,
        long: 'service'
      }),
      address: option({
        type: cmdstring,
        long: 'address'
      }),
      eventName: option({
        type: cmdstring,
        long: 'eventName'
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
    name: 'listener',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db) {
  async function wrapper({
    chain,
    service,
    dryrun
  }: {
    chain?: string
    service?: string
    dryrun?: boolean
  }) {
    let where = ''
    if (chain) {
      const chainId = await chainToId(db, chain)
      where += ` WHERE chainId=${chainId}`
    }
    if (service) {
      if (where.length) {
        where += ' AND '
      } else {
        where += ' WHERE '
      }
      const serviceId = await serviceToId(db, service)
      where += `serviceId=${serviceId}`
    }

    const query = `SELECT * FROM Listener ${where}`
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
    service,
    address,
    eventName,
    dryrun
  }: {
    chain: string
    service: string
    address: string
    eventName: string
    dryrun?: boolean
  }) {
    const chainId = await chainToId(db, chain)
    const serviceId = await serviceToId(db, service)
    const query = `INSERT INTO Listener (chainId, serviceId, address, eventName) VALUES (${chainId}, ${chainId},'${address}', '${eventName}');`

    if (dryrun) {
      console.debug(query)
    } else {
      await db.run(query)
    }
  }
  return wrapper
}

export function removeHandler(db) {
  async function wrapper({ id, dryrun }: { id: number; dryrun?: boolean }) {
    const query = `DELETE FROM Listener WHERE id=${id};`
    if (dryrun) {
      console.debug(query)
    } else {
      await db.run(query)
    }
  }
  return wrapper
}
