import { parseArgs } from 'node:util'
import { SETTINGS_DB_FILE } from '../settings'
import sqlite from 'sqlite3'
import { open } from 'sqlite'

import {
  flag,
  boolean as cmdboolean,
  number as cmdnumber,
  binary,
  optional,
  option,
  subcommands,
  command,
  run,
  string,
  positional
} from 'cmd-ts'

const idOption = option({
  type: cmdnumber,
  long: 'id'
})

const dryrunOption = flag({
  type: cmdboolean,
  long: 'dry-run'
})

const serviceOption = option({
  type: optional(string),
  long: 'service'
})

function buildOption({ name, isOptional }: { name: string; isOptional?: boolean }) {
  return option({
    type: isOptional ? optional(string) : string,
    long: name
  })
}

async function main() {
  const db = await open({
    filename: SETTINGS_DB_FILE,
    driver: sqlite.Database
  })

  // await db.migrate({ force: true }) // FIXME

  const chain = chainSub(db)
  const service = serviceSub(db)
  const listener = listenerSub(db)
  const vrf = vrfSub(db)

  const cli = subcommands({
    name: 'operator',
    cmds: { chain, service, listener, vrf }
  })

  run(binary(cli), process.argv)
}

function chainSub(db) {
  const list = command({
    name: 'list',
    args: {},
    handler: async ({}) => {
      const query = `SELECT * FROM Chain`
      const result = await db.all(query)
      console.log(result)
    }
  })

  const insert = command({
    name: 'insert',
    args: {
      name: option({
        type: string,
        long: 'name'
      }),
      dryrun: dryrunOption
    },
    handler: async ({ name, dryrun }) => {
      const query = `INSERT INTO Chain (name) VALUES ('${name}')`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: async ({ id, dryrun }) => {
      const query = `DELETE FROM Chain WHERE id=${id}`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  return subcommands({
    name: 'chain',
    cmds: { list, insert, remove }
  })
}

function serviceSub(db) {
  const list = command({
    name: 'list',
    args: {},
    handler: async ({}) => {
      const query = `SELECT * FROM Service`
      const result = await db.all(query)
      console.log(result)
    }
  })

  const insert = command({
    name: 'insert',
    args: {
      name: buildOption({ name: 'name' }),
      dryrun: dryrunOption
    },
    handler: async ({ name, dryrun }) => {
      const query = `INSERT INTO Service (name) VALUES ('${name}')`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: async ({ id, dryrun }) => {
      const query = `DELETE FROM Service WHERE id=${id}`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  return subcommands({
    name: 'chain',
    cmds: { list, insert, remove }
  })
}

function listenerSub(db) {
  const list = command({
    name: 'list',
    args: {
      chain: buildOption({ name: 'chain', isOptional: true }),
      service: buildOption({ name: 'service', isOptional: true }),
      dryrun: dryrunOption
    },
    handler: async ({ chain, service, dryrun }) => {
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

      const query = `SELECT id, address, eventName FROM Listener ${where}`
      if (dryrun) {
        console.debug(query)
      } else {
        const result = await db.all(query)
        console.log(result)
      }
    }
  })

  const insert = command({
    name: 'list',
    args: {
      chain: buildOption({ name: 'chain', isOptional: true }),
      service: buildOption({ name: 'service' }),
      address: buildOption({ name: 'address' }),
      eventName: buildOption({ name: 'eventName' }),
      dryrun: dryrunOption
    },
    handler: async ({ chain, service, address, eventName, dryrun }) => {
      const chainResult = await db.get(`SELECT id from Chain WHERE name='${chain}'`)
      const serviceResult = await db.get(`SELECT id from Service WHERE name='${service}'`)
      const query = `INSERT INTO Listener (chainId, serviceId, address, eventName) VALUES (${chainResult.id}, ${serviceResult.id},'${address}', '${eventName}');`

      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: async ({ id, dryrun }) => {
      const query = `DELETE FROM Listener WHERE id=${id};`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  return subcommands({
    name: 'listener',
    cmds: { list, insert, remove }
  })
}

function vrfSub(db) {
  const list = command({
    name: 'list',
    args: {
      chain: buildOption({ name: 'chain', isOptional: true }),
      dryrun: dryrunOption
    },
    handler: async ({ chain, dryrun }) => {
      let where = ''
      if (chain) {
        // where += ' WHERE '
        where += `AND Chain.name = '${chain}'`
      }
      const query = `SELECT VrfKey.id, Chain.name as chain, sk, pk, pk_x, pk_y FROM VrfKey LEFT OUTER JOIN Chain
   ON VrfKey.chainId = Chain.id ${where};`
      if (dryrun) {
        console.debug(query)
      } else {
        const result = await db.all(query)
        console.log(result)
      }
    }
  })

  const insert = command({
    name: 'insert',
    args: {
      chain: buildOption({ name: 'chain' }),
      sk: buildOption({ name: 'sk' }),
      pk: buildOption({ name: 'pk' }),
      pk_x: buildOption({ name: 'pk_x' }),
      pk_y: buildOption({ name: 'pk_y' }),
      dryrun: dryrunOption
    },
    handler: async ({ chain, dryrun, pk, sk, pk_x, pk_y }) => {
      const chainResult = await db.get(`SELECT id from Chain WHERE name='${chain}'`)
      const query = `INSERT INTO VrfKey (chainId, sk, pk, pk_x, pk_y) VALUES (${chainResult.id}, '${sk}', '${pk}', '${pk_x}', '${pk_y}');`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  const remove = command({
    name: 'remove',
    args: {
      id: idOption,
      dryrun: dryrunOption
    },
    handler: async ({ id, dryrun }) => {
      const query = `DELETE FROM VrfKey WHERE id=${id};`
      if (dryrun) {
        console.debug(query)
      } else {
        await db.run(query)
      }
    }
  })

  return subcommands({
    name: 'vrf',
    cmds: { list, insert, remove }
  })
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
