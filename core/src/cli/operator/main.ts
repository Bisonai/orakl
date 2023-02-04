import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { migrateCmd } from './migrate'
import { kvSub } from './kv'
import { adapterSub } from './adapter'
import { aggregatorSub } from './aggregator'
import { openDb } from './utils'
import { buildLogger } from '../../logger'

import { binary, subcommands, run } from 'cmd-ts'

const LOGGER = buildLogger('cli')

async function main() {
  const db = await openDb({})

  const chain = chainSub(db, LOGGER)
  const service = serviceSub(db, LOGGER)
  const listener = listenerSub(db, LOGGER)
  const vrf = vrfSub(db, LOGGER)
  const migrate = migrateCmd(db)
  const kv = kvSub(db, LOGGER)
  const adapter = adapterSub(db, LOGGER)
  const aggregator = aggregatorSub(db, LOGGER)

  const cli = subcommands({
    name: 'operator',
    cmds: { migrate, kv, chain, service, listener, vrf, adapter, aggregator }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  LOGGER.error(error)
  process.exitCode = 1
})
