import { parseArgs } from 'node:util'
import { SETTINGS_DB_FILE } from '../settings'
import sqlite from 'sqlite3'
import { open } from 'sqlite'

import {
  flag,
  boolean as cmdboolean,
  binary,
  optional,
  option,
  subcommands,
  command,
  run,
  string,
  positional
} from 'cmd-ts'

// FIXME move somewhere else
const ALLOWED_CHAINS = ['localhost', 'baobab', 'cypress']

async function main() {
  const db = await open({
    filename: SETTINGS_DB_FILE,
    driver: sqlite.Database
  })
  await db.migrate({ force: true }) // FIXME

  const chain = option({
    type: optional(string),
    long: 'chain'
  })

  const debug = flag({
    type: cmdboolean,
    long: 'debug'
  })

  const listener = command({
    name: 'listener',
    args: {
      chain,
      debug,
      service: option({
        type: optional(string),
        long: 'service'
      })
    },
    handler: async ({ chain, service, debug }) => {
      let where = ''
      if (chain) {
        where += ' WHERE '
        where += `chainId = (SELECT id from Chain WHERE name='${chain}')`
      }
      if (service) {
        if (where.length) {
          where += ' AND '
        }
        where += `serviceId = (SELECT id from Service WHERE name='${service}')`
      }

      const query = `SELECT address, eventName FROM Listener ${where}`
      if (debug) {
        console.debug(query)
      }

      const result = await db.all(query)
      console.log(result)
    }
  })

  const vrf = command({
    name: 'vrf',
    args: {
      chain,
      debug
    },
    handler: async ({ chain, debug }) => {
      let where = ''
      if (chain) {
        where += ' WHERE '
        where += `chainId = (SELECT id from Chain WHERE name='${chain}')`
      }
      const query = `SELECT sk, pk, pk_x, pk_y FROM VrfKey ${where}`
      if (debug) {
        console.debug(query)
      }
      const result = await db.all(query)
      console.log(result)
    }
  })

  const cli = subcommands({
    name: 'operator',
    cmds: { listener, vrf }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
