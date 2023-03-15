#!/usr/bin/env node --no-warnings --experimental-specifier-resolution=node

import os from 'node:os'
import path from 'node:path'
import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { migrateCmd } from './migrate'
import { adapterSub } from './adapter'
import { aggregatorSub } from './aggregator'
import { fetcherSub } from './fetcher'
import { openDb } from './utils'

import { binary, subcommands, run } from 'cmd-ts'

async function main() {
  const ORAKL_DIR = process.env.ORAKL_DIR || path.join(os.homedir(), '.orakl')
  const dbFile = path.join(ORAKL_DIR, 'settings.sqlite')
  const db = await openDb({ dbFile })

  const chain = chainSub()
  const service = serviceSub()
  const listener = listenerSub(db)
  const vrf = vrfSub(db)
  const migrate = migrateCmd(db)
  const adapter = adapterSub()
  const aggregator = aggregatorSub()
  const fetcher = fetcherSub()

  const cli = subcommands({
    name: 'operator',
    cmds: { migrate, chain, service, listener, vrf, adapter, aggregator, fetcher }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
