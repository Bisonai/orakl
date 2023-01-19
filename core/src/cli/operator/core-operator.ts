import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { openDb, dryrunOption, idOption } from './utils'

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
  string
} from 'cmd-ts'

async function main() {
  const db = await openDb()

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

function vrfSub(db) {
  const list = command({
    name: 'list',
    args: {
      chain: buildStringOption({ name: 'chain', isOptional: true }),
      dryrun: dryrunOption
    },
    handler: async ({ chain, dryrun }) => {
      let where = ''
      if (chain) {
        where += `AND Chain.name = '${chain}'`
      }
      const query = `SELECT VrfKey.id, Chain.name as chain, sk, pk, pk_x, pk_y FROM VrfKey INNER JOIN Chain
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
      chain: buildStringOption({ name: 'chain' }),
      sk: buildStringOption({ name: 'sk' }),
      pk: buildStringOption({ name: 'pk' }),
      pk_x: buildStringOption({ name: 'pk_x' }),
      pk_y: buildStringOption({ name: 'pk_y' }),
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
