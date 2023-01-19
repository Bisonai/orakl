import { command, subcommands, option, string as cmdstring } from 'cmd-ts'
import { dryrunOption, idOption, chainOptionalOption, serviceOptionalOption } from './utils'

export function listenerSub(db) {
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
      where += ' WHERE '
      where += `chainId = (SELECT id from Chain WHERE name='${chain}')`
    }
    if (service) {
      if (where.length) {
        where += ' AND '
      } else {
        where += ' WHERE '
      }
      where += `serviceId = (SELECT id from Service WHERE name='${service}')`
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
    const chainResult = await db.get(`SELECT id from Chain WHERE name='${chain}'`)
    const serviceResult = await db.get(`SELECT id from Service WHERE name='${service}'`)
    const query = `INSERT INTO Listener (chainId, serviceId, address, eventName) VALUES (${chainResult.id}, ${serviceResult.id},'${address}', '${eventName}');`

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
