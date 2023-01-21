import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { migrateCmd } from './migrate'
import { kvSub } from './kv'
import { adapterSub } from './adapter'
import { openDb } from './utils'

import { binary, subcommands, run } from 'cmd-ts'

async function main() {
  const db = await openDb({})

  const chain = chainSub(db)
  const service = serviceSub(db)
  const listener = listenerSub(db)
  const vrf = vrfSub(db)
  const migrate = migrateCmd(db)
  const kv = kvSub(db)
  const adapter = adapterSub(db)

  const cli = subcommands({
    name: 'operator',
    cmds: { migrate, kv, chain, service, listener, vrf, adapter }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
