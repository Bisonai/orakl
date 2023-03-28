#!/usr/bin/env node --no-warnings --experimental-specifier-resolution=node

import { command } from 'cmd-ts'
import { chainSub } from './chain'
import { serviceSub } from './service'
import { listenerSub } from './listener'
import { vrfSub } from './vrf'
import { adapterSub } from './adapter'
import { aggregatorSub } from './aggregator'
import { fetcherSub } from './fetcher'
import { reporterSub } from './reporter'

import { binary, subcommands, run } from 'cmd-ts'

async function main() {
  const chain = chainSub()
  const service = serviceSub()
  const listener = listenerSub()
  const vrf = vrfSub()
  const adapter = adapterSub()
  const aggregator = aggregatorSub()
  const fetcher = fetcherSub()
  const reporter = reporterSub()
  const version = command({
    name: 'version',
    args: {},
    handler: () => {
      console.log(`Orakl Network CLI v${process.env.npm_package_version}`)
    }
  })

  const cli = subcommands({
    name: 'operator',
    cmds: { chain, service, listener, vrf, adapter, aggregator, fetcher, reporter, version }
  })

  run(binary(cli), process.argv)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
