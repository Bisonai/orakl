import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { openDb } from './utils'

import { binary, subcommands, run } from 'cmd-ts'

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

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
