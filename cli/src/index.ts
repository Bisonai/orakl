#!/usr/bin/env node --no-warnings --experimental-specifier-resolution=node

import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { adapterSub } from './adapter'
import { aggregatorSub } from './aggregator'
import { fetcherSub } from './fetcher'

import { binary, subcommands, run } from 'cmd-ts'

async function main() {
  const chain = chainSub()
  const service = serviceSub()
  const listener = listenerSub()
  const vrf = vrfSub()
  const adapter = adapterSub()
  const aggregator = aggregatorSub()
  const fetcher = fetcherSub()

  const cli = subcommands({
    name: 'operator',
    cmds: { chain, service, listener, vrf, adapter, aggregator, fetcher }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
