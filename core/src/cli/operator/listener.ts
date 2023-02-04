import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { Logger } from 'pino'
import {
  dryrunOption,
  idOption,
  chainOptionalOption,
  serviceOptionalOption,
  chainToId,
  serviceToId,
  formatResultInsert,
  formatResultRemove
} from './utils'

export function listenerSub(db, logger: Logger) {
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
    handler: listHandler(db, true, logger)
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
    name: 'listener',
    cmds: { list, insert, remove }
  })
}

export function listHandler(db, print?: boolean, logger?: Logger) {
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
    const query = `INSERT INTO Listener (chainId, serviceId, address, eventName) VALUES (${chainId}, ${serviceId},'${address}', '${eventName}');`

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
    const query = `DELETE FROM Listener WHERE id=${id};`
    if (dryrun) {
      logger?.debug(query)
    } else {
      const result = await db.run(query)
      logger?.info(formatResultRemove(result))
    }
  }
  return wrapper
}
